package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// installFakeOps puts an `ops` script first in PATH and returns the log file it
// appends "argv|OPS_ADMIN_PASSWORD" lines to.
func installFakeOps(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	logPath := filepath.Join(dir, "calls.log")
	script := "#!/bin/sh\nprintf '%s|%s\n' \"$*\" \"${OPS_ADMIN_PASSWORD:-}\" >> \"" + logPath + "\"\n" + body
	if err := os.WriteFile(filepath.Join(dir, "ops"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return logPath
}

func readCalls(t *testing.T, logPath string) []string {
	t.Helper()
	data, _ := os.ReadFile(logPath)
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func TestAddOpsUserKeepsPasswordOutOfArgv(t *testing.T) {
	logPath := installFakeOps(t, "exit 0\n")
	if out, err := addOpsUser("demoapp1", "demoapp1@example.com", "s3cret-pw"); err != nil {
		t.Fatalf("addOpsUser: %v %s", err, out)
	}
	calls := readCalls(t, logPath)
	if len(calls) != 1 || calls[0] != "admin adduser demoapp1 demoapp1@example.com --all|s3cret-pw" {
		t.Fatalf("unexpected calls: %q", calls)
	}
}

func TestAddOpsUserFallsBackForOldOps(t *testing.T) {
	// An older ops requires the positional password and answers with usage.
	logPath := installFakeOps(t, `
if [ "$#" -lt 6 ]; then
  echo "Usage:"
  echo "  admin adduser <username> <email> <password> [--all]"
  exit 1
fi
exit 0
`)
	if out, err := addOpsUser("demoapp1", "demoapp1@example.com", "s3cret-pw"); err != nil {
		t.Fatalf("addOpsUser: %v %s", err, out)
	}
	calls := readCalls(t, logPath)
	if len(calls) != 2 || calls[1] != "admin adduser demoapp1 demoapp1@example.com s3cret-pw --all|" {
		t.Fatalf("expected an env attempt then a legacy retry, got %q", calls)
	}
}

func TestAddOpsUserDoesNotRetryOtherFailures(t *testing.T) {
	logPath := installFakeOps(t, "echo boom\nexit 3\n")
	out, err := addOpsUser("demoapp1", "demoapp1@example.com", "s3cret-pw")
	if err == nil || !strings.Contains(string(out), "boom") {
		t.Fatalf("expected the original failure, got err=%v out=%q", err, out)
	}
	if calls := readCalls(t, logPath); len(calls) != 1 {
		t.Fatalf("a non-usage failure must not retry: %q", calls)
	}
}
