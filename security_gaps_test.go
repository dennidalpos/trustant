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

package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"strings"
	"testing"
)

func TestParseSessionKey(t *testing.T) {
	long := strings.Repeat("k", 32)
	encoded := "base64:" + base64.StdEncoding.EncodeToString([]byte(long))
	short := "base64:" + base64.StdEncoding.EncodeToString([]byte("too-short"))

	cases := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"plain 32 bytes", long, false},
		{"base64 32 bytes", encoded, false},
		{"plain too short", "short", true},
		{"base64 too short", short, true},
		{"base64 invalid", "base64:***not-base64***", true},
		{"empty", "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			key, err := parseSessionKey(tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("parseSessionKey(%q) error = %v, wantErr %v", tc.value, err, tc.wantErr)
			}
			if !tc.wantErr && string(key) != long {
				t.Fatalf("parseSessionKey(%q) = %q, want %q", tc.value, key, long)
			}
		})
	}
}

func TestAuthSessionEndpoint(t *testing.T) {
	disabled := &authManager{enabled: false}
	rec := httptest.NewRecorder()
	disabled.handleSession(rec, httptest.NewRequest(http.MethodGet, "/api/auth/session", nil))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Fatalf("disabled session = %d %s", rec.Code, rec.Body.String())
	}

	a := testAuthManager()
	handler := authTestHandler(a)
	cookie, csrf := loginForTest(t, handler, "admin", "correct horse")

	req := authRequest(http.MethodGet, "http://trustant.example.test/api/auth/session", nil)
	req.AddCookie(cookie)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("session status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Enabled  bool   `json:"enabled"`
		Username string `json:"username"`
		CSRF     string `json:"csrf"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode session: %s", err)
	}
	if !body.Enabled || body.Username != "admin" || body.CSRF != csrf {
		t.Fatalf("session body = %+v, want enabled admin with login CSRF", body)
	}

	// Without a session the middleware must refuse before handleSession runs.
	anonymous := httptest.NewRecorder()
	handler.ServeHTTP(anonymous, authRequest(http.MethodGet, "http://trustant.example.test/api/auth/session", nil))
	if anonymous.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous session status = %d, want 401", anonymous.Code)
	}

	// Called directly with an enabled manager but no session in the context.
	direct := httptest.NewRecorder()
	a.handleSession(direct, httptest.NewRequest(http.MethodGet, "/api/auth/session", nil))
	if direct.Code != http.StatusUnauthorized {
		t.Fatalf("contextless session status = %d, want 401", direct.Code)
	}

	post := httptest.NewRecorder()
	a.handleSession(post, httptest.NewRequest(http.MethodPost, "/api/auth/session", nil))
	if post.Code != http.StatusMethodNotAllowed || post.Header().Get("Allow") != http.MethodGet {
		t.Fatalf("POST session = %d Allow=%q, want 405 GET", post.Code, post.Header().Get("Allow"))
	}
}

func TestAuthLogoutRejectsWrongMethodAndDisabled(t *testing.T) {
	disabled := &authManager{enabled: false}
	rec := httptest.NewRecorder()
	disabled.handleLogout(rec, httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("disabled logout = %d, want 404", rec.Code)
	}

	a := testAuthManager()
	rec = httptest.NewRecorder()
	a.handleLogout(rec, httptest.NewRequest(http.MethodGet, "/api/auth/logout", nil))
	if rec.Code != http.StatusMethodNotAllowed || rec.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("GET logout = %d Allow=%q, want 405 POST", rec.Code, rec.Header().Get("Allow"))
	}
}

func TestParseHostnameProtocolAndPort(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "http://trustant.example.test:8910/", nil)
	host, port, proto := parseHostname(req)
	if host != "trustant.example.test" || port != "8910" || proto != "http" {
		t.Fatalf("parseHostname = %q %q %q", host, port, proto)
	}

	req = httptest.NewRequest(http.MethodGet, "http://trustant.example.test/", nil)
	req.Host = "trustant.example.test"
	req.Header.Set("X-Forwarded-Proto", "https")
	host, port, proto = parseHostname(req)
	if host != "trustant.example.test" || port != "" || proto != "https" {
		t.Fatalf("forwarded parseHostname = %q %q %q", host, port, proto)
	}
}

func TestHostnameMiddlewareRoutingAndRedirects(t *testing.T) {
	originalTruacp, originalVite := truacpProxy, viteProxy
	originalWorkspace := WorkspaceDir
	WorkspaceDir = t.TempDir() // unknown labels look up apps here, never in the real workspace
	t.Cleanup(func() {
		truacpProxy, viteProxy = originalTruacp, originalVite
		WorkspaceDir = originalWorkspace
	})
	stub := func(label string) *httputil.ReverseProxy {
		target, _ := url.Parse("http://" + label + ".invalid")
		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			rec.WriteString(label)
			return rec.Result(), nil
		})
		return proxy
	}
	truacpProxy, viteProxy = stub("truacp"), stub("vite")

	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("trustant")) })
	handler := hostnameMiddleware(next)

	serve := func(target string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
		return rec
	}

	for target, want := range map[string]string{
		"http://trustant.example.test/":       "trustant",
		"http://truacp.example.test:8910/x":   "truacp",
		"http://vite.example.test/assets/app": "vite",
	} {
		if rec := serve(target); rec.Code != http.StatusOK || rec.Body.String() != want {
			t.Fatalf("%s = %d %q, want 200 %q", target, rec.Code, rec.Body.String(), want)
		}
	}

	rec := serve("http://127.0.0.1:8910/api/x?y=1")
	if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "http://trustant.127.0.0.1.nip.io:8910/api/x?y=1" {
		t.Fatalf("IP redirect = %d %q", rec.Code, rec.Header().Get("Location"))
	}
	rec = serve("http://localhost/")
	if rec.Code != http.StatusTemporaryRedirect || rec.Header().Get("Location") != "http://trustant.127.0.0.1.nip.io/" {
		t.Fatalf("localhost redirect = %d %q", rec.Code, rec.Header().Get("Location"))
	}

	rec = serve("http://unknown.example.test:8910/")
	if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), "http://trustant.example.test:8910") {
		t.Fatalf("unknown host = %d %q, want 400 with suggested URL", rec.Code, rec.Body.String())
	}
}

func TestHostnameMiddlewareEscapesHostInErrorPage(t *testing.T) {
	originalWorkspace := WorkspaceDir
	WorkspaceDir = t.TempDir()
	t.Cleanup(func() { WorkspaceDir = originalWorkspace })

	handler := hostnameMiddleware(http.NotFoundHandler())
	req := httptest.NewRequest(http.MethodGet, "http://trustant.example.test/", nil)
	req.Host = `x<script>alert(1)</script>"y.ex<b>ample.test`
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	body := rec.Body.String()
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rec.Code)
	}
	for _, raw := range []string{"<script>", "</script>", "<b>", `"y`} {
		if strings.Contains(body, raw) {
			t.Fatalf("error page contains unescaped %q: %s", raw, body)
		}
	}
	if !strings.Contains(body, "&lt;b&gt;ample.test") {
		t.Fatalf("error page missing escaped domain: %s", body)
	}
}
