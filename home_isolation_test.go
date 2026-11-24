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

	home := filepath.Join(root, "home")
	if err := os.MkdirAll(home, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "cannot create the isolated test home:", err)
		return 1
	}
	// git, used by many tests, reads the identity from the global config. Give the
	// isolated home a COPY of the real one: commits made by tests keep working,
	// while `git config --global` (Configure sets user.name and user.email) can
	// no longer rewrite the developer's own identity.
	if realHome, err := os.UserHomeDir(); err == nil {
		if data, err := os.ReadFile(filepath.Join(realHome, ".gitconfig")); err == nil {
			if err := os.WriteFile(filepath.Join(home, ".gitconfig"), data, 0o600); err != nil {
				fmt.Fprintln(os.Stderr, "cannot copy the git configuration:", err)
				return 1
			}
		}
	}
	os.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	os.Setenv("HOME", home)
	os.Setenv("USERPROFILE", home)
	os.Setenv("PI_CODING_AGENT_DIR", filepath.Join(root, "pi-agent"))
	return m.Run()
}
