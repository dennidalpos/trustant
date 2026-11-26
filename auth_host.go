// Copyright 2025-2026 Nuvolaris Inc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Host sessions protect the proxied assistant host (truacp.<domain>) when local
// authentication is on. The browser proves a login on the trustant host, asks
// for a one-time token with POST /api/auth/host-token, and redeems it on the
// proxied host at hostAuthPath. That sets a host-only cookie, so the session
// cookie of the trustant host is never sent to, or shared with, another host.
const (
	hostAuthPath          = "/__trustant/auth"
	hostSessionCookie     = "trustant_host_session"
	hostSecureCookie      = "__Host-trustant_host_session"
	hostTokenTTL          = time.Minute
	hostTokenMaxEntries   = 256
	hostTokenRequestLimit = 1 << 10
)

// gatedHostLabels are the routing labels that require a host session.
var gatedHostLabels = map[string]bool{"truacp": true}

// hostAuth is the manager that gates the proxied hosts. main sets it; it stays
// nil in tests that exercise routing alone, which leaves the hosts open.
var hostAuth *authManager

type hostToken struct {
	sessionID string
	label     string
	expires   time.Time
}

// hostSignature binds a cookie value to one host label, so a value minted for
// one host, or the trustant session cookie, is not accepted on another.
func (a *authManager) hostSignature(label, id string) string {
	mac := hmac.New(sha256.New, a.sessionKey)
	mac.Write([]byte("host|" + label + "|" + id))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *authManager) hostCookieValue(label, id string) string {
	return id + "." + a.hostSignature(label, id)
}

// handleHostToken (POST /api/auth/host-token, authenticated, CSRF-checked by
// the middleware) issues a single-use token for one proxied host.
func (a *authManager) handleHostToken(w http.ResponseWriter, r *http.Request) {
	if !a.enabled {
		http.NotFound(w, r)
		return
	}
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, _ := r.Context().Value(authContextKey{}).(*authSession)
	if session == nil {
		writeAuthJSON(w, http.StatusUnauthorized, map[string]any{"error": "authentication required"})
		return
	}
	var req struct {
		Host string `json:"host"`
	}
	r.Body = http.MaxBytesReader(w, r.Body, hostTokenRequestLimit)
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || !gatedHostLabels[req.Host] {
		writeAuthJSON(w, http.StatusBadRequest, map[string]any{"error": "unknown host"})
		return
	}
	token, err := randomToken(a.random, 32)
	if err != nil {
		writeAuthJSON(w, http.StatusInternalServerError, map[string]any{"error": "failed to create token"})
		return
	}
	now := a.now()
	a.mu.Lock()
	if a.hostTokens == nil {
		a.hostTokens = make(map[string]*hostToken)
	}
	for key, t := range a.hostTokens {
		if !t.expires.After(now) {
			delete(a.hostTokens, key)
		}
	}
	for len(a.hostTokens) >= hostTokenMaxEntries {
		var oldestKey string
		var oldest time.Time
		for key, t := range a.hostTokens {
			if oldestKey == "" || t.expires.Before(oldest) {
				oldestKey, oldest = key, t.expires
			}
		}
		delete(a.hostTokens, oldestKey)
	}
	a.hostTokens[token] = &hostToken{sessionID: session.ID, label: req.Host, expires: now.Add(hostTokenTTL)}
	a.mu.Unlock()
	writeAuthJSON(w, http.StatusOK, map[string]any{"token": token, "path": hostAuthPath})
}

// redeemHostToken consumes a token. It is single use and bound to one label.
func (a *authManager) redeemHostToken(token, label string) (*authSession, bool) {
	now := a.now()
	a.mu.Lock()
	defer a.mu.Unlock()
	t := a.hostTokens[token]
	if t == nil {
		return nil, false
	}
	delete(a.hostTokens, token)
	if t.label != label || !t.expires.After(now) {
		return nil, false
	}
	session := a.sessions[t.sessionID]
	if session == nil || !session.ExpiresAt.After(now) {
		return nil, false
	}
	copy := *session
	return &copy, true
}

func hostCookieOf(r *http.Request) *http.Cookie {
	if c, err := r.Cookie(hostSecureCookie); err == nil {
		return c
	}
	if c, err := r.Cookie(hostSessionCookie); err == nil {
		return c
	}
	return nil
}

// hostSession returns the live session behind the host cookie of r, if any.
func (a *authManager) hostSession(r *http.Request, label string) *authSession {
	cookie := hostCookieOf(r)
	if cookie == nil {
		return nil
	}
	dot := strings.LastIndex(cookie.Value, ".")
	if dot <= 0 {
		return nil
	}
	id, signature := cookie.Value[:dot], cookie.Value[dot+1:]
	if !hmac.Equal([]byte(signature), []byte(a.hostSignature(label, id))) {
		return nil
	}
	now := a.now()
	a.mu.Lock()
	defer a.mu.Unlock()
	session := a.sessions[id]
	if session == nil || !session.ExpiresAt.After(now) {
		return nil
	}
	copy := *session
	return &copy
}

func (a *authManager) setHostCookie(w http.ResponseWriter, r *http.Request, label string, session *authSession) {
	https := requestIsHTTPS(r)
	name := hostSessionCookie
	if https {
		name = hostSecureCookie
	}
	cookie := &http.Cookie{
		Name:     name,
		Value:    a.hostCookieValue(label, session.ID),
		Path:     "/",
		Expires:  session.ExpiresAt,
		MaxAge:   int(session.ExpiresAt.Sub(a.now()).Seconds()),
		HttpOnly: true,
		Secure:   https,
		SameSite: http.SameSiteStrictMode,
	}
	// The assistant is embedded as an iframe of the trustant host. When the two
	// hosts are different sites (for example *.localhost or a public-suffix
	// domain such as nip.io) a Strict cookie is dropped in that third-party
	// frame. Over HTTPS, or on a loopback name that browsers treat as secure,
	// issue a partitioned cookie (CHIPS): it is only sent inside this embedding
	// and never to other sites.
	if https || isLoopbackHostname(r) {
		cookie.Secure = true
		cookie.SameSite = http.SameSiteNoneMode
		cookie.Partitioned = true
	}
	http.SetCookie(w, cookie)
}

// isLoopbackHostname reports whether the request host is localhost or a
// *.localhost name, which browsers treat as a secure context.
func isLoopbackHostname(r *http.Request) bool {
	hostname, _, _ := parseHostname(r)
	hostname = strings.ToLower(hostname)
	return hostname == "localhost" || strings.HasSuffix(hostname, ".localhost")
}

// hostGate wraps the handler of a proxied host. With local authentication off
// it is the handler itself. Otherwise every request needs a host session,
// except the token redemption path.
func (a *authManager) hostGate(label string, next http.Handler) http.Handler {
	if a == nil || !a.enabled || !gatedHostLabels[label] {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == hostAuthPath {
			a.serveHostAuth(w, r, label)
			return
		}
		session := a.hostSession(r, label)
		if session == nil {
			a.hostUnauthorized(w, r, label)
			return
		}
		if isWebSocketUpgrade(r) && !sameUpgradeOrigin(r) {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		stripHostCookie(r)
		next.ServeHTTP(w, r)
	})
}

func (a *authManager) serveHostAuth(w http.ResponseWriter, r *http.Request, label string) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	session, ok := a.redeemHostToken(r.URL.Query().Get("token"), label)
	if !ok {
		a.hostUnauthorized(w, r, label)
		return
	}
	next := r.URL.Query().Get("next")
	if !strings.HasPrefix(next, "/") || strings.HasPrefix(next, "//") || strings.HasPrefix(next, "/\\") || next == hostAuthPath {
		next = "/"
	}
	a.setHostCookie(w, r, label, session)
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	http.Redirect(w, r, next, http.StatusSeeOther)
}

func (a *authManager) hostUnauthorized(w http.ResponseWriter, r *http.Request, label string) {
	hostname, port, protocol := parseHostname(r)
	login := ""
	if dot := strings.Index(hostname, "."); dot > 0 {
		login = protocol + "://trustant." + hostname[dot+1:]
		if port != "" {
			login += ":" + port
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusUnauthorized)
	fmt.Fprint(w, "<!DOCTYPE html><html><head><title>Authentication required</title></head><body><p>Authentication required.")
	if login != "" {
		fmt.Fprintf(w, ` Sign in on <a href="%s">%s</a>.`, html.EscapeString(login), html.EscapeString(login))
	}
	fmt.Fprint(w, "</p></body></html>")
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}

// sameUpgradeOrigin rejects a WebSocket handshake from another origin. A
// handshake without an Origin header comes from a non-browser client, which
// cannot ride on the cookie of a victim, so it is allowed.
func sameUpgradeOrigin(r *http.Request) bool {
	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && strings.EqualFold(u.Host, r.Host)
}

// stripHostCookie removes the host session cookie before the request reaches
// the backend, so the proxied application never sees it.
func stripHostCookie(r *http.Request) {
	cookies := r.Cookies()
	r.Header.Del("Cookie")
	for _, c := range cookies {
		if c.Name == hostSessionCookie || c.Name == hostSecureCookie {
			continue
		}
		r.AddCookie(c)
	}
}
