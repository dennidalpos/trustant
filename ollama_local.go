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
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

// The local Ollama is the ollama.service systemd unit installed next to
// Trustant (start.sh). The UI shows its state and can start or stop it, which
// matters when the same machine also runs another Ollama (for example the one
// of the Windows host reached from WSL) and the two compete for localhost:11434.
const (
	ollamaLocalUnit = "ollama"
	ollamaLocalURL  = "http://127.0.0.1:11434/api/tags"
)

// ollamaLocalStatus is the JSON returned by /api/ollama-local.
type ollamaLocalStatus struct {
	// Supported is true when a loadable ollama.service exists on this host.
	Supported bool `json:"supported"`
	// WSL reports whether Trustant runs inside WSL.
	WSL bool `json:"wsl"`
	// State is the systemd LoadState: loaded, masked, not-found, or "" without systemd.
	State string `json:"state"`
	// Active is true when systemd reports the unit active.
	Active bool `json:"active"`
	// Reachable is true when localhost:11434 answers /api/tags.
	Reachable bool `json:"reachable"`
	// Models is the number of models the local Ollama serves (0 if unreachable).
	Models int    `json:"models"`
	Error  string `json:"error,omitempty"`
}

// Indirections so tests run without systemd, sudo or a network.
var (
	ollamaLocalRun = func(ctx context.Context, name string, args ...string) (string, error) {
		out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
		return strings.TrimSpace(string(out)), err
	}
	ollamaLocalProbe = func(ctx context.Context) (int, bool) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, ollamaLocalURL, nil)
		if err != nil {
			return 0, false
		}
		resp, err := (&http.Client{Timeout: 2 * time.Second}).Do(req)
		if err != nil {
			return 0, false
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return 0, false
		}
		var tags struct {
			Models []json.RawMessage `json:"models"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
			return 0, true
		}
		return len(tags.Models), true
	}
	ollamaLocalGOOS  = runtime.GOOS
	ollamaLocalIsWSL = func() bool {
		if os.Getenv("WSL_DISTRO_NAME") != "" {
			return true
		}
		data, err := os.ReadFile("/proc/sys/kernel/osrelease")
		return err == nil && strings.Contains(strings.ToLower(string(data)), "microsoft")
	}
	ollamaLocalIsRoot = func() bool { return os.Geteuid() == 0 }
)

func readOllamaLocalStatus(ctx context.Context) ollamaLocalStatus {
	st := ollamaLocalStatus{WSL: ollamaLocalIsWSL()}
	if ollamaLocalGOOS != "linux" {
		return st
	}
	state, err := ollamaLocalRun(ctx, "systemctl", "show", "-p", "LoadState", "--value", ollamaLocalUnit)
	if err != nil {
		// No systemd (or no systemctl): nothing to control from here.
		return st
	}
	st.State = state
	st.Supported = state == "loaded"
	if active, _ := ollamaLocalRun(ctx, "systemctl", "is-active", ollamaLocalUnit); active == "active" {
		st.Active = true
	}
	st.Models, st.Reachable = ollamaLocalProbe(ctx)
	if state == "masked" {
		st.Error = "The ollama service is masked on this host; unmask it to control it from here."
	}
	return st
}

// ollamaLocalSystemctl runs `systemctl <verb> ollama`, through `sudo -n` when
// the server is not root, so a missing sudo rule fails fast instead of waiting
// for a password.
func ollamaLocalSystemctl(ctx context.Context, verb string) (string, error) {
	if ollamaLocalIsRoot() {
		return ollamaLocalRun(ctx, "systemctl", verb, ollamaLocalUnit)
	}
	return ollamaLocalRun(ctx, "sudo", "-n", "systemctl", verb, ollamaLocalUnit)
}

// handleOllamaLocal serves the local Ollama state (GET) and starts or stops it
// (POST {"action":"start"|"stop"}).
func handleOllamaLocal(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		json.NewEncoder(w).Encode(readOllamaLocalStatus(r.Context()))
	case http.MethodPost:
		var body struct {
			Action string `json:"action"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<10)).Decode(&body); err != nil ||
			(body.Action != "start" && body.Action != "stop") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": `action must be "start" or "stop"`})
			return
		}
		st := readOllamaLocalStatus(r.Context())
		if !st.Supported {
			w.WriteHeader(http.StatusConflict)
			if st.Error == "" {
				st.Error = "No local Ollama service was found on this host."
			}
			json.NewEncoder(w).Encode(st)
			return
		}
		if out, err := ollamaLocalSystemctl(r.Context(), body.Action); err != nil {
			st = readOllamaLocalStatus(r.Context())
			st.Error = "Could not " + body.Action + " Ollama: " + firstLine(out, err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(st)
			return
		}
		st = waitOllamaLocal(r.Context(), body.Action == "start")
		json.NewEncoder(w).Encode(st)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// waitOllamaLocal polls until the service reaches the wanted state (answering
// when started, inactive when stopped) or the scaled timeout expires, and
// returns the last status read.
func waitOllamaLocal(ctx context.Context, wantRunning bool) ollamaLocalStatus {
	deadline := time.Now().Add(scaledTimeout(15 * time.Second))
	for {
		st := readOllamaLocalStatus(ctx)
		reached := (wantRunning && st.Active && st.Reachable) || (!wantRunning && !st.Active)
		if reached || time.Now().After(deadline) || ctx.Err() != nil {
			return st
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func firstLine(text, fallback string) string {
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return fallback
}
