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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
	"time"
)

const hostTestOrigin = "http://truacp.example.test"

// hostTokenFor logs in on the trustant host and asks for a host token.
func hostTokenFor(t *testing.T, handler http.Handler, cookie *http.Cookie, csrf, host string) (int, string) {
	t.Helper()
	req := authRequest(http.MethodPost, "http://trustant.example.test/api/auth/host-token", []byte(`{"host":"`+host+`"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	req.Header.Set(authCSRFHeader, csrf)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var body struct {
		Token string `json:"token"`
	}
	json.Unmarshal(rec.Body.Bytes(), &body)
	return rec.Code, body.Token
}

func hostRequest(method, path string) *http.Request {
	req := httptest.NewRequest(method, hostTestOrigin+path, nil)
	req.Host = "truacp.example.test"
	req.RemoteAddr = "192.0.2.10:1234"
	return req
}

// redeemForTest turns a token into the host cookie on the gated host.
func redeemForTest(t *testing.T, gate http.Handler, token string) *http.Cookie {
	t.Helper()
	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, hostRequest(http.MethodGet, hostAuthPath+"?token="+url.QueryEscape(token)+"&next=%2Fchat"))
	if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/chat" {
		t.Fatalf("redeem = %d location=%q", rec.Code, rec.Header().Get("Location"))
	}
	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("redeem cookies = %d, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != hostSessionCookie || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Domain != "" || c.Path != "/" {
		t.Fatalf("unexpected host cookie: %#v", c)
	}
	return c
}

func TestHostGateIsTransparentWithAuthDisabled(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	for _, a := range []*authManager{nil, {enabled: false}} {
		rec := httptest.NewRecorder()
		a.hostGate("truacp", next).ServeHTTP(rec, hostRequest(http.MethodGet, "/"))
		if rec.Code != http.StatusTeapot {
			t.Fatalf("disabled gate = %d, want pass-through", rec.Code)
		}
	}
	// A label that is not gated is never wrapped, even with auth on.
	rec := httptest.NewRecorder()
	testAuthManager().hostGate("vite", next).ServeHTTP(rec, hostRequest(http.MethodGet, "/"))
	if rec.Code != http.StatusTeapot {
		t.Fatalf("ungated label = %d, want pass-through", rec.Code)
	}
}

func TestHostSessionFlow(t *testing.T) {
	a := testAuthManager()
	handler := authTestHandler(a)
	var seenCookie string
	backend := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenCookie = r.Header.Get("Cookie")
		w.Write([]byte("truacp"))
	})
	gate := a.hostGate("truacp", backend)

	// No cookie: the assistant host is closed, with a pointer to the login host.
	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, hostRequest(http.MethodGet, "/"))
	if rec.Code != http.StatusUnauthorized || !strings.Contains(rec.Body.String(), "http://trustant.example.test") {
		t.Fatalf("anonymous = %d %q", rec.Code, rec.Body.String())
	}

	cookie, csrf := loginForTest(t, handler, "admin", "correct horse")

	// The trustant session cookie value does not open the assistant host.
	req := hostRequest(http.MethodGet, "/")
	req.AddCookie(&http.Cookie{Name: hostSessionCookie, Value: cookie.Value})
	rec = httptest.NewRecorder()
	gate.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("trustant cookie value on the assistant host = %d, want 401", rec.Code)
	}

	// The token request itself needs the CSRF header and a known host.
	noCSRF := authRequest(http.MethodPost, "http://trustant.example.test/api/auth/host-token", []byte(`{"host":"truacp"}`))
	noCSRF.AddCookie(cookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, noCSRF)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("token without CSRF = %d, want 403", rec.Code)
	}
	if code, _ := hostTokenFor(t, handler, cookie, csrf, "vite"); code != http.StatusBadRequest {
		t.Fatalf("token for an ungated host = %d, want 400", code)
	}
	anon := authRequest(http.MethodPost, "http://trustant.example.test/api/auth/host-token", []byte(`{"host":"truacp"}`))
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, anon)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("token without session = %d, want 401", rec.Code)
	}

	code, token := hostTokenFor(t, handler, cookie, csrf, "truacp")
	if code != http.StatusOK || token == "" {
		t.Fatalf("token = %d %q", code, token)
	}
	host := redeemForTest(t, gate, token)

	// The token is single use.
	rec = httptest.NewRecorder()
	gate.ServeHTTP(rec, hostRequest(http.MethodGet, hostAuthPath+"?token="+url.QueryEscape(token)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("token replay = %d, want 401", rec.Code)
	}

	// The host cookie opens the host and never reaches the backend.
	req = hostRequest(http.MethodGet, "/")
	req.AddCookie(host)
	req.AddCookie(&http.Cookie{Name: "app", Value: "keep"})
	rec = httptest.NewRecorder()
	gate.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "truacp" {
		t.Fatalf("authenticated = %d %q", rec.Code, rec.Body.String())
	}
	if strings.Contains(seenCookie, hostSessionCookie) || !strings.Contains(seenCookie, "app=keep") {
		t.Fatalf("backend cookie header = %q", seenCookie)
	}

	// The host cookie is not a trustant session.
	req = authRequest(http.MethodGet, "http://trustant.example.test/api/read", nil)
	req.AddCookie(&http.Cookie{Name: authSessionCookie, Value: host.Value})
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("host cookie value on the trustant host = %d, want 401", rec.Code)
	}

	// Logging out closes the assistant host too.
	logout := authRequest(http.MethodPost, "http://trustant.example.test/api/auth/logout", nil)
	logout.AddCookie(cookie)
	logout.Header.Set(authCSRFHeader, csrf)
	handler.ServeHTTP(httptest.NewRecorder(), logout)
	req = hostRequest(http.MethodGet, "/")
	req.AddCookie(host)
	rec = httptest.NewRecorder()
	gate.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("host cookie after logout = %d, want 401", rec.Code)
	}
}

func TestHostTokenExpiryAndLabel(t *testing.T) {
	a := testAuthManager()
	now := time.Now()
	a.now = func() time.Time { return now }
	handler := authTestHandler(a)
	gate := a.hostGate("truacp", http.NotFoundHandler())
	cookie, csrf := loginForTest(t, handler, "admin", "correct horse")

	_, expired := hostTokenFor(t, handler, cookie, csrf, "truacp")
	now = now.Add(hostTokenTTL + time.Second)
	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, hostRequest(http.MethodGet, hostAuthPath+"?token="+url.QueryEscape(expired)))
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expired token = %d, want 401", rec.Code)
	}

	_, token := hostTokenFor(t, handler, cookie, csrf, "truacp")
	if _, ok := a.redeemHostToken(token, "vite"); ok {
		t.Fatal("a token for truacp was accepted for another host")
	}
	if _, ok := a.redeemHostToken(token, "truacp"); ok {
		t.Fatal("a token that failed once must not be redeemable afterwards")
	}
}

func TestHostAuthRedirectIsLocal(t *testing.T) {
	a := testAuthManager()
	handler := authTestHandler(a)
	gate := a.hostGate("truacp", http.NotFoundHandler())
	cookie, csrf := loginForTest(t, handler, "admin", "correct horse")
	for _, next := range []string{"//evil.test/", "https://evil.test/", "/\\evil.test", hostAuthPath} {
		_, token := hostTokenFor(t, handler, cookie, csrf, "truacp")
		rec := httptest.NewRecorder()
		gate.ServeHTTP(rec, hostRequest(http.MethodGet, hostAuthPath+"?token="+url.QueryEscape(token)+"&next="+url.QueryEscape(next)))
		if rec.Code != http.StatusSeeOther || rec.Header().Get("Location") != "/" {
			t.Fatalf("next=%q -> %d %q, want 303 to /", next, rec.Code, rec.Header().Get("Location"))
		}
	}
}

func TestHostGateRejectsForeignWebSocketOrigin(t *testing.T) {
	a := testAuthManager()
	handler := authTestHandler(a)
	gate := a.hostGate("truacp", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ws")) }))
	cookie, csrf := loginForTest(t, handler, "admin", "correct horse")
	_, token := hostTokenFor(t, handler, cookie, csrf, "truacp")
	host := redeemForTest(t, gate, token)

	for origin, want := range map[string]int{
		"":                         http.StatusOK,
		hostTestOrigin:             http.StatusOK,
		"http://evil.example.test": http.StatusForbidden,
	} {
		req := hostRequest(http.MethodGet, "/socket")
		req.Header.Set("Upgrade", "websocket")
		if origin != "" {
			req.Header.Set("Origin", origin)
		}
		req.AddCookie(host)
		rec := httptest.NewRecorder()
		gate.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Fatalf("websocket origin %q = %d, want %d", origin, rec.Code, want)
		}
	}
	// Without the host cookie the upgrade never reaches the backend.
	req := hostRequest(http.MethodGet, "/socket")
	req.Header.Set("Upgrade", "websocket")
	rec := httptest.NewRecorder()
	gate.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous websocket = %d, want 401", rec.Code)
	}
}

func TestHostnameMiddlewareGatesTruacpAndLeavesOtherHosts(t *testing.T) {
	originalTruacp, originalVite, originalAuth := truacpProxy, viteProxy, hostAuth
	originalWorkspace := WorkspaceDir
	WorkspaceDir = t.TempDir()
	t.Cleanup(func() {
		truacpProxy, viteProxy, hostAuth = originalTruacp, originalVite, originalAuth
		WorkspaceDir = originalWorkspace
	})
	stub := func(label string) *httputil.ReverseProxy {
		target, _ := url.Parse("http://" + label + ".invalid")
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = roundTripFunc(func(*http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			rec.WriteString(label)
			return rec.Result(), nil
		})
		return proxy
	}
	truacpProxy, viteProxy = stub("truacp"), stub("vite")
	hostAuth = testAuthManager()

	handler := hostnameMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("trustant")) }))
	serve := func(target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec
	}
	if rec := serve("http://truacp.example.test:8910/x"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("truacp without a host session = %d, want 401", rec.Code)
	}
	if rec := serve("http://vite.example.test/x"); rec.Code != http.StatusOK || rec.Body.String() != "vite" {
		t.Fatalf("vite = %d %q, want it unchanged", rec.Code, rec.Body.String())
	}
	if rec := serve("http://trustant.example.test/"); rec.Code != http.StatusOK || rec.Body.String() != "trustant" {
		t.Fatalf("trustant = %d %q", rec.Code, rec.Body.String())
	}
}

func TestHostCookieIsPartitionedWhereEmbeddingNeedsIt(t *testing.T) {
	a := testAuthManager()
	session := &authSession{ID: "s1", ExpiresAt: a.now().Add(time.Hour)}
	cases := []struct {
		name, host, proto string
		embedded          bool
	}{
		{"plain http on a public name stays strict", "truacp.example.test", "", false},
		{"loopback name is embeddable", "truacp.localhost:8910", "", true},
		{"https is embeddable", "truacp.example.test", "https", true},
	}
	for _, tc := range cases {
		req := hostRequest(http.MethodGet, "/")
		req.Host = tc.host
		if tc.proto != "" {
			req.Header.Set("X-Forwarded-Proto", tc.proto)
		}
		rec := httptest.NewRecorder()
		a.setHostCookie(rec, req, "truacp", session)
		c := rec.Result().Cookies()[0]
		if tc.embedded {
			if !c.Secure || c.SameSite != http.SameSiteNoneMode || !c.Partitioned {
				t.Errorf("%s: want Secure+None+Partitioned, got %#v", tc.name, c)
			}
		} else if c.Secure || c.SameSite != http.SameSiteStrictMode || c.Partitioned {
			t.Errorf("%s: want Strict, got %#v", tc.name, c)
		}
		if !c.HttpOnly || c.Domain != "" || c.Path != "/" {
			t.Errorf("%s: unexpected attributes %#v", tc.name, c)
		}
	}
}
