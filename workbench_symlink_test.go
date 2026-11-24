package main

import (
	"os"
	"path/filepath"
	"testing"
)

// plantSymlink makes <dir>/<name> a symlink to a victim file outside the
// workbench and returns the victim path.
func plantSymlink(t *testing.T, dir, name string) string {
	t.Helper()
	victim := filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(victim, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(dir, name)); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	return victim
}

func assertUntouched(t *testing.T, victim string, err error) {
	t.Helper()
	if got, _ := os.ReadFile(victim); string(got) != "original" {
		t.Fatalf("target outside the workbench was overwritten: %q", got)
	}
	if err == nil {
		t.Fatal("expected an error when the target is a symlink")
	}
}

func TestWriteEnvFileDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := plantSymlink(t, dir, ".env")
	err := writeEnvFile(filepath.Join(dir, ".env"), map[string]string{"OPS_PASSWORD": "secret"}, []string{"OPS_PASSWORD"})
	assertUntouched(t, victim, err)
}

func TestWriteManagedInstructionFileDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := plantSymlink(t, dir, "AGENTS.md")
	assertUntouched(t, victim, writeManagedInstructionFile(dir, "AGENTS.md"))
}

func TestEnsureManagedGitignoreDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := plantSymlink(t, dir, ".gitignore")
	_, err := ensureManagedGitignore(dir)
	assertUntouched(t, victim, err)
}

func TestWriteEnvDistBindingsDoesNotFollowSymlink(t *testing.T) {
	dir := t.TempDir()
	victim := plantSymlink(t, dir, ".env.dist")
	_, err := writeEnvDistBindings(filepath.Join(dir, ".env.dist"), []EnvBinding{{Name: "A", Pattern: "b"}})
	assertUntouched(t, victim, err)
}

func TestSaveSharedEnvDoesNotFollowSymlink(t *testing.T) {
	old := WorkbenchDir
	WorkbenchDir = t.TempDir()
	t.Cleanup(func() { WorkbenchDir = old })
	app := "demoapp1"
	if err := os.MkdirAll(filepath.Join(WorkbenchDir, app), 0o755); err != nil {
		t.Fatal(err)
	}
	victim := plantSymlink(t, filepath.Join(WorkbenchDir, app), sharedEnvFileName)
	assertUntouched(t, victim, saveSharedEnv(app, map[string]string{"x__A": "1"}))
}
