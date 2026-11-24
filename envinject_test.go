package main

import (
	"encoding/json"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const injected = "x\nOPS_APIHOST=http://evil.example.test"

func assertNotWritten(t *testing.T, path string) {
	t.Helper()
	if data, err := os.ReadFile(path); err == nil {
		t.Fatalf("%s must not be written, got %q", path, data)
	}
}

func TestWriteEnvFileRejectsLineInjection(t *testing.T) {
	dir := t.TempDir()
	for name, vars := range map[string]map[string]string{
		"newline in value": {"FOO": injected},
		"newline in name":  {"FOO\nBAR": "x"},
		"nul in value":     {"FOO": "a\x00b"},
		"cr in value":      {"FOO": "a\rOPS_USER=b"},
	} {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "_"))
		if err := writeEnvFile(path, vars, nil); err == nil {
			t.Errorf("%s: expected an error", name)
		}
		assertNotWritten(t, path)
	}
}

func TestWriteEnvDistBindingsRejectsLineInjection(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env.dist")
	_, err := writeEnvDistBindings(path, []EnvBinding{{Name: "A", Pattern: "db*\nOPS_PASSWORD=*"}})
	if err == nil {
		t.Fatal("expected an error for a pattern containing a newline")
	}
	assertNotWritten(t, path)
}

func TestSaveSharedEnvRejectsLineInjection(t *testing.T) {
	root := t.TempDir()
	old := WorkbenchDir
	WorkbenchDir = root
	t.Cleanup(func() { WorkbenchDir = old })
	if err := os.MkdirAll(filepath.Join(root, "demoapp1"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := saveSharedEnv("demoapp1", map[string]string{"demoapp1__DB": "<.a>\notherapp__DB=<.b>"})
	if err == nil {
		t.Fatal("expected an error for a value containing a newline")
	}
	assertNotWritten(t, filepath.Join(root, "demoapp1", sharedEnvFileName))
}

func TestPostAppConfigRejectsInjectedValuesAndNames(t *testing.T) {
	for name, vars := range map[string][]EnvVar{
		"newline in dev value":  {{Name: "FOO", DevValue: injected}},
		"newline in prod value": {{Name: "FOO", ProdValue: injected}},
		"invalid name":          {{Name: "BAD NAME", DevValue: "x"}},
		"name with equals":      {{Name: "A=B", DevValue: "x"}},
	} {
		workbench := envDistTestApp(t, "demo", map[string]string{"ALPHA": "a"}, nil)
		body, _ := json.Marshal(map[string]interface{}{"vars": vars})
		rec := httptest.NewRecorder()
		handlePostAppConfig(rec, httptest.NewRequest("POST", "/api/appconfig/demo", strings.NewReader(string(body))),
			"demo", filepath.Join(WorkspaceDir, "workspace", "demo"))
		if rec.Code != 400 {
			t.Errorf("%s: status = %d, want 400 (%s)", name, rec.Code, rec.Body)
		}
		cfg, err := loadWorkspaceConfig()
		if err != nil {
			t.Fatal(err)
		}
		if app := cfg.Apps["demo"]; app == nil || app.Development["ALPHA"] != "a" || app.Development["FOO"] != "" {
			t.Errorf("%s: stored config changed: %+v", name, app)
		}
		assertNotWritten(t, filepath.Join(workbench, ".env.evil"))
	}
}

func TestSharedAndImportsHandlersRejectControlCharacters(t *testing.T) {
	workbench := envDistTestApp(t, "demoapp", nil, nil)
	sharedBody, _ := json.Marshal(map[string]interface{}{"vars": []sharedVarRequest{{Name: "DEMOAPP__DB", Path: "a\nOTHERAPP__DB=<.b>"}}})
	rec := httptest.NewRecorder()
	handleSharedSave(rec, httptest.NewRequest("POST", "/api/shared/demoapp", strings.NewReader(string(sharedBody))), "demoapp")
	if rec.Code != 400 {
		t.Errorf("shared save: status = %d, want 400 (%s)", rec.Code, rec.Body)
	}
	assertNotWritten(t, filepath.Join(workbench, sharedEnvFileName))

	// Positive control: the same request with a clean path passes validation,
	// so the 400 above is caused by the control character and nothing else.
	cleanBody, _ := json.Marshal(map[string]interface{}{"vars": []sharedVarRequest{{Name: "DEMOAPP__DB", Path: "a"}}})
	rec = httptest.NewRecorder()
	handleSharedSave(rec, httptest.NewRequest("POST", "/api/shared/demoapp", strings.NewReader(string(cleanBody))), "demoapp")
	if rec.Code == 400 {
		t.Errorf("clean shared save rejected: %s", rec.Body)
	}
	_ = os.Remove(filepath.Join(workbench, sharedEnvFileName))

	importBody, _ := json.Marshal(map[string]interface{}{"bindings": []EnvBinding{{Name: "DBURL", Pattern: "db*\nOPS_PASSWORD=*"}}})
	rec = httptest.NewRecorder()
	handleImportsSave(rec, httptest.NewRequest("POST", "/api/imports/demoapp", strings.NewReader(string(importBody))), "demoapp")
	if rec.Code != 400 {
		t.Errorf("imports save: status = %d, want 400 (%s)", rec.Code, rec.Body)
	}
	assertNotWritten(t, filepath.Join(workbench, ".env.dist"))
}

func TestValidEnvValuesStillRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	vars := map[string]string{"URL": "https://x.example.test/?a=b&c=d", "REF": "${{SHARED}}", "SPACED": "two words"}
	if err := writeEnvFile(path, vars, []string{"URL", "REF", "SPACED"}); err != nil {
		t.Fatal(err)
	}
	got := parseEnvFile(path)
	for k, v := range vars {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
}
