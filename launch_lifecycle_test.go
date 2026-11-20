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
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"
)

func TestManagedRuntimeHealthyForAppRequiresOwnedProcessGroupAndBothPorts(t *testing.T) {
	origWorkbench := WorkbenchDir
	WorkbenchDir = t.TempDir()
	t.Cleanup(func() { WorkbenchDir = origWorkbench })

	left, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen left: %s", err)
	}
	defer left.Close()
	right, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen right: %s", err)
	}
	defer right.Close()

	if err := os.WriteFile(filepath.Join(WorkbenchDir, "current"), []byte("trutest"), 0644); err != nil {
		t.Fatalf("write current: %s", err)
	}
	if err := os.WriteFile(filepath.Join(WorkbenchDir, "pgid"), []byte(strconv.Itoa(syscall.Getpgrp())), 0644); err != nil {
		t.Fatalf("write pgid: %s", err)
	}

	leftPort := left.Addr().(*net.TCPAddr).Port
	rightPort := right.Addr().(*net.TCPAddr).Port
	if !managedRuntimeHealthyForApp("trutest", leftPort, rightPort) {
		t.Fatal("expected matching app, live process group, and both listeners to be reusable")
	}
	if managedRuntimeHealthyForApp("another-app", leftPort, rightPort) {
		t.Fatal("must not reuse a runtime owned by another app")
	}

	if err := right.Close(); err != nil {
		t.Fatalf("close right listener: %s", err)
	}
	if managedRuntimeHealthyForApp("trutest", leftPort, rightPort) {
		t.Fatal("must not reuse a runtime when either shared service is unavailable")
	}
}

func TestRedeployOnlyRestartsTheActiveAppsWatcher(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is required for the TruACP and preview fixtures")
	}
	root := isolateSharedWorkspace(t)
	const active = "viteapp"
	const inactive = "otherapp"
	activePath := makeWorkbench(t, active)
	makeWorkbench(t, inactive)
	activePath, err = canonicalWorkbenchPath(active)
	if err != nil {
		t.Fatal(err)
	}

	writeFixture := func(name, content string, mode os.FileMode) string {
		t.Helper()
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(content), mode); err != nil {
			t.Fatal(err)
		}
		return path
	}
	leaderScript := writeFixture("truacp.cjs", "setInterval(() => {}, 1000);\n", 0600)
	watcherScript := writeFixture("vite-fixture.cjs", `
const [port, label] = process.argv.slice(2);
require("node:http").createServer((_, res) => res.end(label))
    .listen(Number(port), "127.0.0.1");
`, 0600)
	writeFixture("ops", `#!/bin/sh
set -eu
printf '%s %s\n' "${PWD##*/}" "$*" >> "$TEST_REDEPLOY_OPS_LOG"
case "$*" in
  "ide deploy") ;;
  "action list") echo "fixture/actions" ;;
  "ide devel --fast") exec "$TEST_REDEPLOY_NODE" "$TEST_REDEPLOY_WATCHER" "$TEST_REDEPLOY_PORT" restarted ;;
  *) exit 2 ;;
esac
`, 0755)
	opsLog := filepath.Join(root, "ops.log")
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_REDEPLOY_NODE", node)
	t.Setenv("TEST_REDEPLOY_WATCHER", watcherScript)
	t.Setenv("TEST_REDEPLOY_OPS_LOG", opsLog)

	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_REDEPLOY_PORT", strconv.Itoa(port))

	leader := exec.Command(node, leaderScript, "--port", strconv.Itoa(truacpPort), "--dir", activePath)
	leader.Dir = activePath
	leader.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := leader.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := leader.Process.Pid
	t.Cleanup(func() {
		_ = killPgid(pgid)
		_ = leader.Wait()
	})
	oldWatcher := exec.Command(node, watcherScript, strconv.Itoa(port), "original")
	oldWatcher.Dir = activePath
	oldWatcher.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: pgid}
	if err := oldWatcher.Start(); err != nil {
		t.Fatal(err)
	}
	oldWatcherExited := make(chan struct{})
	go func() {
		_ = oldWatcher.Wait()
		close(oldWatcherExited)
	}()
	if err := waitForHTTP(port, 3*time.Second); err != nil {
		t.Fatal(err)
	}
	if err := writePgid(pgid); err != nil {
		t.Fatal(err)
	}

	redeploy := func(app string) string {
		rec := httptest.NewRecorder()
		serveRedeploy(rec, httptest.NewRequest(http.MethodGet, "/api/redeploy?name="+app, nil), port)
		if rec.Code != http.StatusOK {
			t.Fatalf("redeploy %s: HTTP %d: %s", app, rec.Code, rec.Body)
		}
		return rec.Body.String()
	}
	assertPreview := func(want string) {
		t.Helper()
		client := &http.Client{Timeout: time.Second}
		resp, err := client.Get("http://127.0.0.1:" + strconv.Itoa(port))
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil || string(body) != want {
			t.Fatalf("preview = %q, %v, want %q", body, err, want)
		}
	}
	// Matching the active-app marker alone must not authorize a stale PGID.
	for _, marker := range []string{active, inactive} {
		if err := writeCurrentApp(marker); err != nil {
			t.Fatal(err)
		}
		events := redeploy(inactive)
		if !strings.Contains(events, "event: error") || strings.Contains(events, "event: status") {
			t.Fatalf("redeploy should reject ownership before doing any work: %s", events)
		}
		assertPreview("original")
	}
	if _, err := os.Stat(opsLog); !os.IsNotExist(err) {
		t.Fatalf("ownership rejection ran ops: %v", err)
	}

	if err := writeCurrentApp(active); err != nil {
		t.Fatal(err)
	}
	events := redeploy(active)
	if !strings.Contains(events, "event: done\ndata: fixture/actions") || strings.Contains(events, "event: error") {
		t.Fatalf("active redeploy failed: %s", events)
	}
	assertPreview("restarted")
	select {
	case <-oldWatcherExited:
	default:
		t.Error("the original watcher did not exit")
	}
	if !managedRuntimeOwnsGroup(active, activePath, pgid) {
		t.Fatal("TruACP fixture stopped during redeploy")
	}
	commands, err := os.ReadFile(opsLog)
	if err != nil {
		t.Fatal(err)
	}
	if want := "viteapp ide deploy\nviteapp action list\nviteapp ide devel --fast\n"; string(commands) != want {
		t.Errorf("ops commands = %q, want %q", commands, want)
	}
}

func TestRuntimeLifecycleLockSerializesOverlappingOperations(t *testing.T) {
	releaseFirst := lockRuntimeLifecycle("test first")
	firstReleased := false
	defer func() {
		if !firstReleased {
			releaseFirst()
		}
	}()

	entered := make(chan struct{})
	done := make(chan struct{})
	go func() {
		releaseSecond := lockRuntimeLifecycle("test second")
		close(entered)
		releaseSecond()
		close(done)
	}()

	select {
	case <-entered:
		t.Fatal("second lifecycle operation entered while the first still held the lock")
	case <-time.After(100 * time.Millisecond):
	}

	releaseFirst()
	firstReleased = true
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("second lifecycle operation did not proceed after the first released the lock")
	}
}
