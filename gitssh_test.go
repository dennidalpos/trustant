package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestGitSSHCommandVerifiesHostKeys(t *testing.T) {
	home := t.TempDir()
	cmd := gitSSHCommand(home)
	if strings.Contains(cmd, "StrictHostKeyChecking=no") {
		t.Fatalf("host key checking is disabled: %s", cmd)
	}
	if !strings.Contains(cmd, "StrictHostKeyChecking=accept-new") {
		t.Fatalf("missing accept-new: %s", cmd)
	}
	known := filepath.Join(home, ".ssh", "known_hosts_trustant")
	if !strings.Contains(cmd, "UserKnownHostsFile="+known) {
		t.Fatalf("missing dedicated known hosts file %s: %s", known, cmd)
	}
	if !strings.Contains(cmd, "-i "+filepath.Join(home, ".ssh", "id_ed25519")) || !strings.Contains(cmd, "IdentitiesOnly=yes") {
		t.Fatalf("identity options changed: %s", cmd)
	}
}

func TestGitCommandUsesVerifiedSSHCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	cmd := gitCommand(t.TempDir(), "status")
	var got string
	for _, entry := range cmd.Env {
		if strings.HasPrefix(entry, "GIT_SSH_COMMAND=") {
			got = strings.TrimPrefix(entry, "GIT_SSH_COMMAND=")
		}
	}
	if got != gitSSHCommand(home) {
		t.Fatalf("GIT_SSH_COMMAND = %q, want %q", got, gitSSHCommand(home))
	}
}
