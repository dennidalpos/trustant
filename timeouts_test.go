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
	"testing"
	"time"
)

func TestScaledTimeout(t *testing.T) {
	cases := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{"unset keeps the default", "", 30 * time.Second},
		{"factor two doubles", "2", 60 * time.Second},
		{"fraction shortens", "0.5", 15 * time.Second},
		{"malformed keeps the default", "slow", 30 * time.Second},
		{"zero keeps the default", "0", 30 * time.Second},
		{"negative keeps the default", "-3", 30 * time.Second},
		{"huge value is capped", "1000", 300 * time.Second},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Setenv(timeoutFactorEnv, c.value)
			if got := scaledTimeout(30 * time.Second); got != c.want {
				t.Fatalf("scaledTimeout(30s) with %q = %s, want %s", c.value, got, c.want)
			}
		})
	}
}
