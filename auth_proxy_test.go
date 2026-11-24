package main

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func mustProxies(t *testing.T, value string) []*net.IPNet {
	t.Helper()
	nets, err := parseTrustedProxies(value)
	if err != nil {
		t.Fatal(err)
	}
	return nets
}

func TestParseTrustedProxies(t *testing.T) {
	if nets, err := parseTrustedProxies(""); err != nil || len(nets) != 0 {
		t.Fatalf("empty value: %v %v", nets, err)
	}
	nets, err := parseTrustedProxies(" 192.0.2.10 , 10.0.0.0/8 ,2001:db8::1")
	if err != nil || len(nets) != 3 {
		t.Fatalf("valid list: %v %v", nets, err)
	}
	for _, bad := range []string{"not-an-ip", "10.0.0.0/33", "192.0.2.10,,"} {
		if _, err := parseTrustedProxies(bad); err == nil {
			t.Fatalf("%q should be rejected", bad)
		}
	}
}

func TestClientKeyHonoursForwardedForOnlyFromTrustedProxy(t *testing.T) {
	a := testAuthManager()
	a.trustedProxies = mustProxies(t, "192.0.2.10, 203.0.113.0/24")
	req := func(peer string, xff ...string) *http.Request {
		r := authRequest(http.MethodPost, "http://trustant.example.test/api/auth/login", nil)
		r.RemoteAddr = peer + ":4000"
		for _, v := range xff {
			r.Header.Add("X-Forwarded-For", v)
		}
		return r
	}
	cases := []struct {
		name string
		r    *http.Request
		want string
	}{
		{"untrusted peer cannot spoof", req("198.51.100.99", "198.51.100.1"), "198.51.100.99"},
		{"trusted peer, one client", req("192.0.2.10", "198.51.100.1"), "198.51.100.1"},
		{"chain, right-most untrusted wins", req("192.0.2.10", "198.51.100.7, 198.51.100.1, 203.0.113.5"), "198.51.100.1"},
		{"spoofed left entry is ignored", req("192.0.2.10", "10.9.9.9", "198.51.100.1"), "198.51.100.1"},
		{"no header falls back to peer", req("192.0.2.10"), "192.0.2.10"},
		{"malformed header falls back to peer", req("192.0.2.10", "garbage"), "192.0.2.10"},
	}
	for _, tc := range cases {
		if got := a.clientKey(tc.r); got != tc.want {
			t.Errorf("%s: clientKey = %q, want %q", tc.name, got, tc.want)
		}
	}
	// Without configuration forwarded headers are never used.
	plain := testAuthManager()
	if got := plain.clientKey(req("192.0.2.10", "198.51.100.1")); got != "192.0.2.10" {
		t.Errorf("default must use the transport peer, got %q", got)
	}
}

func TestLoginLimiterSeparatesClientsBehindTrustedProxy(t *testing.T) {
	a := testAuthManager()
	a.trustedProxies = mustProxies(t, "192.0.2.10")
	handler := authTestHandler(a)
	login := func(client string) int {
		r := authRequest(http.MethodPost, "http://trustant.example.test/api/auth/login", []byte(`{"username":"admin","password":"wrong"}`))
		r.Header.Set("X-Forwarded-For", client)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, r)
		return rec.Code
	}
	for i := 0; i < defaultLoginMaxFails; i++ {
		login("198.51.100.1")
	}
	if got := login("198.51.100.1"); got != http.StatusTooManyRequests {
		t.Fatalf("abusive client = %d, want 429", got)
	}
	if got := login("198.51.100.2"); got != http.StatusUnauthorized {
		t.Fatalf("other client behind the same proxy = %d, want 401 (not locked out)", got)
	}
}
