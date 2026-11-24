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
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// TestMain keeps the whole suite away from the developer's real home.
//
// Several handlers (Configure, launch, runtime files) write under
// os.UserHomeDir(): ~/.pi/agent for Pi and ~/.config/trustant for the runtime.
// A test that forgot to relocate them used to overwrite the live Pi provider
// and default model of whoever ran `go test` — in the demo VM that broke the
// agent panel. Tests that need a specific home still call t.Setenv, which
// overrides these defaults for their own duration.
func TestMain(m *testing.M) {
	os.Exit(runIsolated(m))
}

func runIsolated(m *testing.M) int {
	root, err := os.MkdirTemp("", "trustant-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "cannot create the isolated test home:", err)
		return 1
	}
	defer os.RemoveAll(root)

	// git, used by many tests, reads the identity from the global config; keep
	// pointing it at the real one so commits made by tests still work.
	if os.Getenv("GIT_CONFIG_GLOBAL") == "" {
		if realHome, err := os.UserHomeDir(); err == nil {
			gitConfig := filepath.Join(realHome, ".gitconfig")
			if _, err := os.Stat(gitConfig); err == nil {
				os.Setenv("GIT_CONFIG_GLOBAL", gitConfig)
			}
		}
	}

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "cannot create the isolated test home:", err)
		return 1
	}
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)
	os.Setenv("PI_CODING_AGENT_DIR", filepath.Join(root, "pi-agent"))
	return m.Run()
}
