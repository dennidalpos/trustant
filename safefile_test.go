package main

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// symlinkFixture prepares a workbench app plus a file outside it that must
// never be modified through the app directory.
func symlinkFixture(t *testing.T) (appDir, outside string) {
	t.Helper()
	old := WorkbenchDir
	WorkbenchDir = t.TempDir()
	t.Cleanup(func() { WorkbenchDir = old })
	appDir = filepath.Join(WorkbenchDir, "demoapp1")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("git", "-C", appDir, "init", "-q").CombinedOutput(); err != nil {
		t.Skipf("git unavailable: %v %s", err, out)
	}
	outside = filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(outside, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	return appDir, outside
}

func TestHandleMemoryDoesNotFollowSymlink(t *testing.T) {
	appDir, outside := symlinkFixture(t)
	if err := os.Symlink(outside, filepath.Join(appDir, "AGENTS.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	rec := httptest.NewRecorder()
	handleMemory(rec, httptest.NewRequest("POST", "/api/memory/demoapp1", strings.NewReader(`{"content":"pwned"}`)))
	if got, _ := os.ReadFile(outside); string(got) != "original" {
		t.Fatalf("target outside the workbench was overwritten: %q", got)
	}
	if rec.Code == http.StatusOK {
		t.Fatalf("expected an error status, got %d", rec.Code)
	}
}

func TestHandleUploadDoesNotFollowSymlink(t *testing.T) {
	appDir, outside := symlinkFixture(t)
	if err := os.MkdirAll(filepath.Join(appDir, "upload"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(appDir, "upload", "x.txt")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	_ = mw.WriteField("name", "demoapp1")
	fw, _ := mw.CreateFormFile("file", "x.txt")
	_, _ = fw.Write([]byte("pwned"))
	_ = mw.Close()
	req := httptest.NewRequest("POST", "/api/upload", &body)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec := httptest.NewRecorder()
	handleUpload(rec, req)
	if got, _ := os.ReadFile(outside); string(got) != "original" {
		t.Fatalf("target outside the workbench was overwritten: %q", got)
	}
	if rec.Code == http.StatusOK {
		t.Fatalf("expected an error status, got %d", rec.Code)
	}
}
