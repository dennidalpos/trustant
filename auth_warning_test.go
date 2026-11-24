package main

import (
	"strings"
	"testing"
)

func TestAuthWarning(t *testing.T) {
	for name, a := range map[string]*authManager{"nil": nil, "disabled": {enabled: false}} {
		w := authWarning(a)
		for _, want := range []string{"8910", "TRUSTANT_AUTH_MODE=local", "all interfaces"} {
			if !strings.Contains(w, want) {
				t.Errorf("%s: warning %q does not mention %q", name, w, want)
			}
		}
	}
	if w := authWarning(testAuthManager()); w != "" {
		t.Errorf("auth enabled must not warn, got %q", w)
	}
}
