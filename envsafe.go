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
	"strings"
)

// checkEnvField rejects characters that would end a line in a .env-style file.
// Every writer of those files serializes `name=value\n`, so a newline, carriage
// return or NUL inside a name, value or pattern would add lines the validation
// in front of the writer never saw.
func checkEnvField(kind, value string) error {
	if strings.ContainsAny(value, "\n\r\x00") {
		return fmt.Errorf("%s must not contain line breaks or NUL characters", kind)
	}
	return nil
}

// checkEnvName also requires a name a shell and a .env file will accept.
func checkEnvName(name string) error {
	if !predefinedEnvNamePattern.MatchString(name) {
		return fmt.Errorf("invalid variable name %q", name)
	}
	return nil
}
