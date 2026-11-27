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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runShK3sReadyBlock extracts the k3s readiness loop of run.sh (step 2b) and
// runs it against a fake `kube` so recovery, persistence and an absent
// namespace can be exercised without a cluster.
func runShK3sReadyBlock(t *testing.T, fakeKube string) (string, int) {
	t.Helper()
	bash, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not available")
	}
	content, err := os.ReadFile("run.sh")
	if err != nil {
		t.Fatal(err)
	}
	script := string(content)
	start := strings.Index(script, "K3S_READY_TIMEOUT=")
	if start < 0 {
		t.Fatal("run.sh lost its k3s readiness loop")
	}
	end := strings.Index(script[start:], "\ndone\n")
	if end < 0 {
		t.Fatal("run.sh k3s readiness loop is not terminated")
	}
	block := script[start : start+end+len("\ndone\n")]
	dir := t.TempDir()
	harness := "count=0\nkube() {\n  count=$((count+1))\n" + fakeKube + "\n}\n" +
		"K3S_READY_TIMEOUT=3\nK3S_READY_INTERVAL=1\n" + block + "echo REACHED\n"
	path := filepath.Join(dir, "h.sh")
	if err := os.WriteFile(path, []byte(harness), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(bash, path)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatal(err)
	}
	return string(out), code
}

func TestRunShToleratesTransientK3sRestart(t *testing.T) {
	// count is not shared across the $(...) subshell, so use a file.
	fake := `  f="$PWD/.n"; n=$(( $(cat "$f" 2>/dev/null || echo 0) + 1 )); echo $n > "$f"
  if [ "$n" -lt 3 ]; then echo "The connection to the server was refused" >&2; return 1; fi
  echo ok`
	out, code := runShK3sReadyBlock(t, fake)
	if code != 0 || !strings.Contains(out, "REACHED") {
		t.Fatalf("a brief k3s outage must be tolerated, got exit %d: %s", code, out)
	}
}

func TestRunShReportsLastErrorWhenK3sStaysDown(t *testing.T) {
	out, code := runShK3sReadyBlock(t, `  echo "connection refused" >&2; return 1`)
	if code != 1 || strings.Contains(out, "REACHED") {
		t.Fatalf("a persistent outage must fail, got exit %d: %s", code, out)
	}
	if !strings.Contains(out, "last kubectl error") || !strings.Contains(out, "connection refused") {
		t.Fatalf("the last kubectl error must be reported: %s", out)
	}
}

func TestRunShFailsFastOnAbsentNamespace(t *testing.T) {
	out, code := runShK3sReadyBlock(t,
		`  echo 'Error from server (NotFound): namespaces "openserverless" not found' >&2; return 1`)
	if code != 1 || strings.Contains(out, "REACHED") || !strings.Contains(out, "namespace missing") {
		t.Fatalf("an absent namespace must fail at once, got exit %d: %s", code, out)
	}
}
