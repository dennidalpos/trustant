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
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// fakeOllamaLocal replaces systemctl, sudo and the HTTP probe with an in-memory
// unit whose state follows start and stop calls.
type fakeOllamaLocal struct {
	loadState string
	active    bool
	models    int
	calls     []string
	failVerb  string
}

func (f *fakeOllamaLocal) install(t *testing.T, root bool) {
	t.Helper()
	prevRun, prevProbe, prevGOOS, prevWSL, prevRoot := ollamaLocalRun, ollamaLocalProbe, ollamaLocalGOOS, ollamaLocalIsWSL, ollamaLocalIsRoot
	t.Cleanup(func() {
		ollamaLocalRun, ollamaLocalProbe, ollamaLocalGOOS, ollamaLocalIsWSL, ollamaLocalIsRoot = prevRun, prevProbe, prevGOOS, prevWSL, prevRoot
	})
	ollamaLocalGOOS = "linux"
	ollamaLocalIsWSL = func() bool { return true }
	ollamaLocalIsRoot = func() bool { return root }
	ollamaLocalRun = func(_ context.Context, name string, args ...string) (string, error) {
		f.calls = append(f.calls, name+" "+strings.Join(args, " "))
		if name == "sudo" {
			args = args[2:] // drop "-n systemctl"
		}
		switch args[0] {
		case "show":
			return f.loadState, nil
		case "is-active":
			if f.active {
				return "active", nil
			}
			return "inactive", errors.New("exit status 3")
		case "start", "stop":
			if f.failVerb == args[0] {
				return "sudo: a password is required", errors.New("exit status 1")
			}
			f.active = args[0] == "start"
			return "", nil
		}
		return "", errors.New("unexpected " + args[0])
	}
	ollamaLocalProbe = func(context.Context) (int, bool) {
		if f.active {
			return f.models, true
		}
		return 0, false
	}
}

func ollamaLocalRequest(t *testing.T, method, body string) (*httptest.ResponseRecorder, ollamaLocalStatus) {
	t.Helper()
	req := httptest.NewRequest(method, "/api/ollama-local", strings.NewReader(body))
	rec := httptest.NewRecorder()
	handleOllamaLocal(rec, req)
	var st ollamaLocalStatus
	_ = json.NewDecoder(rec.Body).Decode(&st)
	return rec, st
}

func TestOllamaLocalStatusReportsStateAndModels(t *testing.T) {
	f := &fakeOllamaLocal{loadState: "loaded", active: true, models: 6}
	f.install(t, true)
	rec, st := ollamaLocalRequest(t, http.MethodGet, "")
	if rec.Code != http.StatusOK || !st.Supported || !st.Active || !st.Reachable || st.Models != 6 || !st.WSL {
		t.Fatalf("status = %d %+v", rec.Code, st)
	}
}

func TestOllamaLocalStartAndStopFollowTheUnit(t *testing.T) {
	t.Setenv(timeoutFactorEnv, "0.01")
	f := &fakeOllamaLocal{loadState: "loaded", models: 3}
	f.install(t, false)

	rec, st := ollamaLocalRequest(t, http.MethodPost, `{"action":"start"}`)
	if rec.Code != http.StatusOK || !st.Active || !st.Reachable || st.Models != 3 {
		t.Fatalf("start = %d %+v", rec.Code, st)
	}
	if !strings.Contains(strings.Join(f.calls, "|"), "sudo -n systemctl start ollama") {
		t.Fatalf("non-root start must use sudo -n, calls: %v", f.calls)
	}

	rec, st = ollamaLocalRequest(t, http.MethodPost, `{"action":"stop"}`)
	if rec.Code != http.StatusOK || st.Active || st.Reachable {
		t.Fatalf("stop = %d %+v", rec.Code, st)
	}
}

func TestOllamaLocalRootSkipsSudo(t *testing.T) {
	t.Setenv(timeoutFactorEnv, "0.01")
	f := &fakeOllamaLocal{loadState: "loaded"}
	f.install(t, true)
	ollamaLocalRequest(t, http.MethodPost, `{"action":"start"}`)
	for _, c := range f.calls {
		if strings.HasPrefix(c, "sudo") {
			t.Fatalf("root must not call sudo: %v", f.calls)
		}
	}
}

func TestOllamaLocalRefusesWhenUnitMaskedOrMissing(t *testing.T) {
	for _, state := range []string{"masked", "not-found"} {
		f := &fakeOllamaLocal{loadState: state}
		f.install(t, true)
		rec, st := ollamaLocalRequest(t, http.MethodPost, `{"action":"start"}`)
		if rec.Code != http.StatusConflict || st.Supported || st.Error == "" {
			t.Fatalf("%s: %d %+v", state, rec.Code, st)
		}
		for _, c := range f.calls {
			if strings.Contains(c, " start ") {
				t.Fatalf("%s: start must not run: %v", state, f.calls)
			}
		}
	}
}

func TestOllamaLocalReportsSudoFailure(t *testing.T) {
	f := &fakeOllamaLocal{loadState: "loaded", failVerb: "start"}
	f.install(t, false)
	rec, st := ollamaLocalRequest(t, http.MethodPost, `{"action":"start"}`)
	if rec.Code != http.StatusInternalServerError || !strings.Contains(st.Error, "password is required") {
		t.Fatalf("sudo failure = %d %+v", rec.Code, st)
	}
}

func TestOllamaLocalRejectsBadInput(t *testing.T) {
	f := &fakeOllamaLocal{loadState: "loaded"}
	f.install(t, true)
	for _, body := range []string{`{"action":"restart"}`, `{}`, `nope`} {
		if rec, _ := ollamaLocalRequest(t, http.MethodPost, body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%q = %d, want 400", body, rec.Code)
		}
	}
	if rec, _ := ollamaLocalRequest(t, http.MethodPut, ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT = %d, want 405", rec.Code)
	}
}

func TestOllamaLocalUnsupportedOffLinux(t *testing.T) {
	f := &fakeOllamaLocal{loadState: "loaded"}
	f.install(t, true)
	ollamaLocalGOOS = "windows"
	if _, st := ollamaLocalRequest(t, http.MethodGet, ""); st.Supported {
		t.Fatalf("windows must be unsupported: %+v", st)
	}
}
