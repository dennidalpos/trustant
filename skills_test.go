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
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setSkillsTestEnv(t *testing.T, defaultRepo, allowed string) {
	t.Helper()
	originalDefault, originalWorkbench := OpsSkills, WorkbenchDir
	OpsSkills = defaultRepo
	WorkbenchDir = t.TempDir()
	t.Setenv("OPS_SKILLS_ALLOWED", allowed)
	t.Cleanup(func() { OpsSkills, WorkbenchDir = originalDefault, originalWorkbench })
}

func TestCheckSkillsSource(t *testing.T) {
	setSkillsTestEnv(t, "trustable-ai/skills", "team/skills, Other/Repo")

	cases := []struct {
		name      string
		repo      string
		confirmed string
		want      error
		invalid   bool
	}{
		{"default", "trustable-ai/skills", "", nil, false},
		{"default case-insensitive", "Trustable-AI/Skills", "", nil, false},
		{"allow-listed", "team/skills", "", nil, false},
		{"allow-listed with spaces", "other/repo", "", nil, false},
		{"unlisted needs confirmation", "evil/skills", "", errSkillsSourceNotConfirmed, false},
		{"unlisted wrong confirmation", "evil/skills", "team/skills", errSkillsSourceNotConfirmed, false},
		{"unlisted confirmed", "evil/skills", "evil/skills", nil, false},
		{"option-like value", "--upload-pack=x/y", "--upload-pack=x/y", nil, true},
		{"url instead of owner/name", "https://evil.example/x", "https://evil.example/x", nil, true},
		{"empty", "", "", nil, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkSkillsSource(tc.repo, tc.confirmed)
			switch {
			case tc.invalid:
				if err == nil || errors.Is(err, errSkillsSourceNotConfirmed) {
					t.Fatalf("checkSkillsSource(%q) = %v, want validation error", tc.repo, err)
				}
			case !errors.Is(err, tc.want) || (tc.want == nil && err != nil):
				t.Fatalf("checkSkillsSource(%q, %q) = %v, want %v", tc.repo, tc.confirmed, err, tc.want)
			}
		})
	}
}

func TestAgentEnvCannotExtendSkillsAllowList(t *testing.T) {
	setSkillsTestEnv(t, "trustable-ai/skills", "")
	app := "demoapp1"
	dir := filepath.Join(WorkbenchDir, app)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("OPS_SKILLS=evil/skills\nOPS_SKILLS_ALLOWED=evil/skills\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := getSkillsRepo(app); got != "evil/skills" {
		t.Fatalf("getSkillsRepo = %q, want evil/skills", got)
	}
	if err := setupSkills(app, ""); !errors.Is(err, errSkillsSourceNotConfirmed) {
		t.Fatalf("setupSkills = %v, want errSkillsSourceNotConfirmed", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".agents")); !os.IsNotExist(err) {
		t.Fatalf(".agents must not be created for an unconfirmed source: %v", err)
	}
	if ensureSkills(app) {
		t.Fatal("ensureSkills installed skills from an unconfirmed source")
	}
}

func TestSkillsUpdateRequiresConfirmationForUnlistedSource(t *testing.T) {
	setSkillsTestEnv(t, "trustable-ai/skills", "")
	app := "demoapp1"
	skills := filepath.Join(WorkbenchDir, app, ".agents", "skills", "keep")
	if err := os.MkdirAll(skills, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(WorkbenchDir, app, ".env"), []byte("OPS_SKILLS=evil/skills\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	handleSkillsUpdate(rec, httptest.NewRequest(http.MethodPost, "/api/skills/"+app+"/update", nil), app)
	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body %s", rec.Code, rec.Body.String())
	}
	var body struct {
		Error                string `json:"error"`
		Source               string `json:"source"`
		RequiresConfirmation bool   `json:"requires_confirmation"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %s", err)
	}
	if body.Source != "evil/skills" || !body.RequiresConfirmation || body.Error == "" {
		t.Fatalf("body = %+v", body)
	}
	if _, err := os.Stat(skills); err != nil {
		t.Fatalf("installed skills were removed by a refused update: %v", err)
	}

	// A confirmation for a different source does not approve this one.
	rec = httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/skills/"+app+"/update", strings.NewReader(`{"confirm_source":"other/skills"}`))
	handleSkillsUpdate(rec, req, app)
	if rec.Code != http.StatusConflict {
		t.Fatalf("wrong confirmation status = %d, want 409", rec.Code)
	}

	// An invalid source is a client error, not a confirmable one.
	if err := os.WriteFile(filepath.Join(WorkbenchDir, app, ".env"), []byte("OPS_SKILLS=--upload-pack=x/y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec = httptest.NewRecorder()
	handleSkillsUpdate(rec, httptest.NewRequest(http.MethodPost, "/api/skills/"+app+"/update", nil), app)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid source status = %d, want 400", rec.Code)
	}
}
