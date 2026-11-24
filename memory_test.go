package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestHandleMemoryRoundTrip(t *testing.T) {
	old := WorkbenchDir
	WorkbenchDir = t.TempDir()
	t.Cleanup(func() { WorkbenchDir = old })
	if err := os.MkdirAll(filepath.Join(WorkbenchDir, "demoapp1"), 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", filepath.Join(WorkbenchDir, "demoapp1"), "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git unavailable: %v %s", err, out)
	}
	do := func(method, path, body string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handleMemory(rec, httptest.NewRequest(method, path, strings.NewReader(body)))
		return rec
	}
	if rec := do("GET", "/api/memory/demoapp1", ""); rec.Code != 200 || !strings.Contains(rec.Body.String(), `"content":""`) {
		t.Fatalf("empty GET: %d %s", rec.Code, rec.Body)
	}
	if rec := do("POST", "/api/memory/demoapp1", `{"content":"# Notes\n"}`); rec.Code != http.StatusOK {
		t.Fatalf("POST: %d %s", rec.Code, rec.Body)
	}
	if b, _ := os.ReadFile(filepath.Join(WorkbenchDir, "demoapp1", "AGENTS.md")); string(b) != "# Notes\n" {
		t.Fatalf("file content %q", b)
	}
	if rec := do("GET", "/api/memory/demoapp1", ""); !strings.Contains(rec.Body.String(), "# Notes") {
		t.Fatalf("GET after POST: %s", rec.Body)
	}
	if rec := do("GET", "/api/memory/bad", ""); rec.Code != 400 {
		t.Fatalf("invalid name: %d", rec.Code)
	}
	if rec := do("GET", "/api/memory/missingapp1", ""); rec.Code != 404 {
		t.Fatalf("missing: %d", rec.Code)
	}
	if rec := do("PUT", "/api/memory/demoapp1", ""); rec.Code != 405 {
		t.Fatalf("method: %d", rec.Code)
	}
}
