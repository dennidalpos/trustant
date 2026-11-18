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
	"strings"
	"testing"
)

// WHY: spec/17-proxy.md documents the streaming timeout chain (GH-204).
// The public proxy keeps one-hour idle limits for slow AI responses; changing
// them silently would shorten streams without any other test noticing.
func TestProxyScriptKeepsOneHourStreamTimeouts(t *testing.T) {
	content, err := os.ReadFile("proxy.sh")
	if err != nil {
		t.Fatalf("read proxy.sh: %s", err)
	}
	for _, required := range []string{
		"proxy_read_timeout   3600s;",
		"proxy_send_timeout   3600s;",
		"proxy_buffering off;",
	} {
		if !strings.Contains(string(content), required) {
			t.Fatalf("proxy.sh is missing %q (see spec/17-proxy.md, streaming timeout chain)", required)
		}
	}
}
