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
	"strconv"
	"strings"
	"time"
)

// Waits that depend on how fast the machine is (a model loading into memory,
// a dev server compiling, a port opening) get a generous default so a slow PC
// is not reported as broken. TRUSTANT_TIMEOUT_FACTOR stretches all of them
// further: 2 doubles every such wait, 1 (the default) keeps them as they are.
const (
	timeoutFactorEnv = "TRUSTANT_TIMEOUT_FACTOR"
	timeoutFactorMax = 10.0
)

// timeoutFactor reads TRUSTANT_TIMEOUT_FACTOR. A missing, malformed or
// non-positive value means 1; a value above the maximum is capped so a typo
// cannot turn a wait into hours.
func timeoutFactor() float64 {
	raw := strings.TrimSpace(os.Getenv(timeoutFactorEnv))
	if raw == "" {
		return 1
	}
	factor, err := strconv.ParseFloat(raw, 64)
	if err != nil || factor <= 0 {
		return 1
	}
	if factor > timeoutFactorMax {
		return timeoutFactorMax
	}
	return factor
}

// scaledTimeout returns d multiplied by the configured timeout factor.
func scaledTimeout(d time.Duration) time.Duration {
	return time.Duration(float64(d) * timeoutFactor())
}
