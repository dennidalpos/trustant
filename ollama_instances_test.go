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
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Tests never probe the real network: host detection is off unless a test
// switches it on with stubHostDetection.
func init() {
	ollamaIsWSL = func() bool { return false }
	ollamaLimaHostResolves = func() bool { return false }
	ollamaHostProbe = func(context.Context, string) bool { return false }
}

func stubHostDetection(t *testing.T, wsl, lima bool, gateway string, answers bool) {
	t.Helper()
	oldWSL, oldLima, oldGw, oldProbe := ollamaIsWSL, ollamaLimaHostResolves, ollamaDefaultGateway, ollamaHostProbe
	t.Cleanup(func() {
		ollamaIsWSL, ollamaLimaHostResolves, ollamaDefaultGateway, ollamaHostProbe = oldWSL, oldLima, oldGw, oldProbe
	})
	ollamaIsWSL = func() bool { return wsl }
	ollamaLimaHostResolves = func() bool { return lima }
	ollamaDefaultGateway = func() string { return gateway }
	ollamaHostProbe = func(context.Context, string) bool { return answers }
}

func ollamaInstancesCall(t *testing.T, method, body string) (*httptest.ResponseRecorder, ollamaInstancesState) {
	t.Helper()
	rec := httptest.NewRecorder()
	handleOllamaInstances(rec, httptest.NewRequest(method, "/api/ollama-instances", strings.NewReader(body)))
	var st ollamaInstancesState
	_ = json.NewDecoder(rec.Body).Decode(&st)
	return rec, st
}

func TestOllamaInstancesSaveListSelectAndDelete(t *testing.T) {
	withPredefinedEnvWorkspace(t, `{"provider":"ollama","base_url":"http://localhost:11434/v1","api_key":"dummy"}`)

	rec, st := ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"name":"Cluster A","url":"http://10.0.0.5:11434/v1/","api_key":"s3cret"}}`)
	if rec.Code != http.StatusOK || len(st.Instances) != 1 {
		t.Fatalf("save = %d %+v", rec.Code, st)
	}
	inst := st.Instances[0]
	if inst.ID != "cluster-a" || inst.URL != "http://10.0.0.5:11434" || !inst.KeySet {
		t.Fatalf("instance = %+v", inst)
	}
	// The key is in a 0600 file and nowhere else.
	info, err := os.Stat(ollamaInstanceKeyPath("cluster-a"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Fatalf("key file: %v %v", info, err)
	}
	stored, _ := os.ReadFile(WorkspaceDir + "/trustant.json")
	if strings.Contains(string(stored), "s3cret") {
		t.Fatalf("the key leaked into trustant.json: %s", stored)
	}

	rec, st = ollamaInstancesCall(t, http.MethodPost, `{"action":"select","id":"cluster-a"}`)
	if rec.Code != http.StatusOK || st.Active != "cluster-a" || st.BaseURL != "http://10.0.0.5:11434/v1" {
		t.Fatalf("select = %d %+v", rec.Code, st)
	}
	cfg, _ := loadTrustantConfig()
	if root, own := resolveOllamaRoot(cfg); root != "http://10.0.0.5:11434" || !own {
		t.Fatalf("resolveOllamaRoot = %q %v", root, own)
	}
	if got := effectiveAPIKey(cfg); got != "s3cret" {
		t.Fatalf("effective key = %q", got)
	}
	if cfg.APIKey != "dummy" {
		t.Fatalf("api_key must stay dummy, got %q", cfg.APIKey)
	}

	// Back to the default: the local Ollama and the dummy key.
	rec, st = ollamaInstancesCall(t, http.MethodPost, `{"action":"select","id":""}`)
	cfg, _ = loadTrustantConfig()
	if rec.Code != http.StatusOK || st.Active != "" || cfg.BaseURL != localOllamaV1URL {
		t.Fatalf("default = %d %+v base=%q", rec.Code, st, cfg.BaseURL)
	}
	if _, own := resolveOllamaRoot(cfg); own {
		t.Fatalf("default must not be own host")
	}

	// Deleting the active instance falls back to the default and removes the key.
	ollamaInstancesCall(t, http.MethodPost, `{"action":"select","id":"cluster-a"}`)
	rec, st = ollamaInstancesCall(t, http.MethodPost, `{"action":"delete","id":"cluster-a"}`)
	cfg, _ = loadTrustantConfig()
	if rec.Code != http.StatusOK || len(st.Instances) != 0 || st.Active != "" || cfg.BaseURL != localOllamaV1URL {
		t.Fatalf("delete = %d %+v base=%q", rec.Code, st, cfg.BaseURL)
	}
	if _, err := os.Stat(ollamaInstanceKeyPath("cluster-a")); !os.IsNotExist(err) {
		t.Fatalf("key file must be removed: %v", err)
	}
	if rec, _ = ollamaInstancesCall(t, http.MethodPost, `{"action":"delete","id":"cluster-a"}`); rec.Code != http.StatusOK {
		t.Fatalf("delete must be idempotent, got %d", rec.Code)
	}
}

func TestOllamaInstancesValidation(t *testing.T) {
	withPredefinedEnvWorkspace(t, `{"provider":"ollama"}`)
	ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"name":"One","url":"http://a:11434"}}`)

	for name, body := range map[string]string{
		"empty name":     `{"action":"save","instance":{"name":" ","url":"http://a:11434"}}`,
		"bad scheme":     `{"action":"save","instance":{"name":"X","url":"ftp://a:11434"}}`,
		"path":           `{"action":"save","instance":{"name":"X","url":"http://a:11434/api"}}`,
		"credentials":    `{"action":"save","instance":{"name":"X","url":"http://u:p@a:11434"}}`,
		"duplicate name": `{"action":"save","instance":{"name":"one","url":"http://b:11434"}}`,
		"bad action":     `{"action":"nope"}`,
		"not json":       `nope`,
	} {
		if rec, _ := ollamaInstancesCall(t, http.MethodPost, body); rec.Code != http.StatusBadRequest {
			t.Fatalf("%s = %d, want 400", name, rec.Code)
		}
	}
	if rec, _ := ollamaInstancesCall(t, http.MethodPost, `{"action":"select","id":"ghost"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("unknown id = %d, want 404", rec.Code)
	}
	if rec, _ := ollamaInstancesCall(t, http.MethodPut, ""); rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("PUT = %d, want 405", rec.Code)
	}
}

func TestOllamaInstancesSelectNeedsTheOllamaProvider(t *testing.T) {
	withPredefinedEnvWorkspace(t, `{"provider":"private","base_url":"http://x:1/v1","api_key":"k"}`)
	ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"name":"A","url":"http://a:11434"}}`)
	if rec, _ := ollamaInstancesCall(t, http.MethodPost, `{"action":"select","id":"a"}`); rec.Code != http.StatusConflict {
		t.Fatalf("select under private = %d, want 409", rec.Code)
	}
	cfg, _ := loadTrustantConfig()
	if cfg.BaseURL != "http://x:1/v1" || cfg.OllamaActive != "" {
		t.Fatalf("the Private AI endpoint was touched: %+v", cfg)
	}
}

func TestOllamaInstancesKeyEditing(t *testing.T) {
	withPredefinedEnvWorkspace(t, `{"provider":"ollama"}`)
	ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"name":"A","url":"http://a:11434","api_key":"k1"}}`)
	// Saving without a key keeps it; a new key replaces it; clear_key removes it.
	_, st := ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"id":"a","name":"A2","url":"http://a2:11434"}}`)
	if !st.Instances[0].KeySet || readOllamaInstanceKey("a") != "k1" || st.Instances[0].Name != "A2" {
		t.Fatalf("key must survive a rename: %+v", st)
	}
	ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"id":"a","name":"A2","url":"http://a2:11434","api_key":"k2"}}`)
	if readOllamaInstanceKey("a") != "k2" {
		t.Fatalf("key not replaced")
	}
	_, st = ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"id":"a","name":"A2","url":"http://a2:11434","clear_key":true}}`)
	if st.Instances[0].KeySet || readOllamaInstanceKey("a") != "" {
		t.Fatalf("key not cleared: %+v", st)
	}
	if rec, _ := ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"id":"ghost","name":"G","url":"http://g:1"}}`); rec.Code != http.StatusNotFound {
		t.Fatalf("editing an unknown id = %d, want 404", rec.Code)
	}
}

func TestOllamaInstancesSurviveSaveConfiguration(t *testing.T) {
	withPredefinedEnvWorkspace(t, `{"provider":"ollama","base_url":"http://localhost:11434/v1","api_key":"dummy"}`)
	ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"name":"A","url":"http://a:11434"}}`)
	ollamaInstancesCall(t, http.MethodPost, `{"action":"select","id":"a"}`)

	// The configure page posts its in-memory form, which does not carry the instances.
	body := `{"provider":"ollama","base_url":"http://a:11434/v1","api_key":"dummy"}`
	handlePostConfiguration(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/configuration", strings.NewReader(body)))

	cfg, err := loadWorkspaceConfig()
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.OllamaInstances) != 1 || cfg.OllamaActive != "a" {
		t.Fatalf("Save & Configure erased the instances: %+v", cfg)
	}
}

func TestDiscoverModelsReadsASavedInstanceWithItsKey(t *testing.T) {
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		w.Write([]byte(`{"data":[{"id":"m1"}]}`))
	}))
	defer server.Close()

	// Provider "private" because the loopback guard of discover-models only
	// applies to "ollama" and the test server listens on 127.0.0.1.
	withPredefinedEnvWorkspace(t, `{"provider":"private"}`)
	body, _ := json.Marshal(map[string]interface{}{"action": "save", "instance": map[string]string{"name": "A", "url": server.URL, "api_key": "tok"}})
	ollamaInstancesCall(t, http.MethodPost, string(body))

	rec := httptest.NewRecorder()
	handleDiscoverModels(rec, httptest.NewRequest(http.MethodPost, "/api/discover-models", strings.NewReader(`{"instance_id":"a"}`)))
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), "m1") || gotAuth != "Bearer tok" {
		t.Fatalf("discover = %d %s auth=%q", rec.Code, rec.Body.String(), gotAuth)
	}
	rec = httptest.NewRecorder()
	handleDiscoverModels(rec, httptest.NewRequest(http.MethodPost, "/api/discover-models", strings.NewReader(`{"instance_id":"ghost"}`)))
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unknown instance = %d, want 404", rec.Code)
	}
}

func TestParseDefaultGateway(t *testing.T) {
	route := "Iface\tDestination\tGateway\tFlags\tRefCnt\tUse\tMetric\tMask\n" +
		"eth0\t00000000\t01E01BAC\t0003\t0\t0\t0\t00000000\n" +
		"eth0\t00E01BAC\t00000000\t0001\t0\t0\t0\t00F0FFFF\n"
	if got := parseDefaultGateway(route); got != "172.27.224.1" {
		t.Fatalf("gateway = %q, want 172.27.224.1", got)
	}
	for _, in := range []string{"", "Iface\tDestination\tGateway\n", "eth0\t00E01BAC\t00000000\n"} {
		if got := parseDefaultGateway(in); got != "" {
			t.Fatalf("parseDefaultGateway(%q) = %q, want empty", in, got)
		}
	}
}

func TestDetectHostOllama(t *testing.T) {
	ctx := context.Background()

	stubHostDetection(t, true, false, "172.27.224.1", true)
	got := detectHostOllama(ctx, &trustantConfig{})
	if got == nil || got.URL != "http://172.27.224.1:11434" || got.Source != "WSL host" {
		t.Fatalf("WSL = %+v", got)
	}
	// A saved instance with the same URL hides the suggestion.
	saved := &trustantConfig{OllamaInstances: []ollamaInstance{{ID: "w", Name: "W", URL: "http://172.27.224.1:11434"}}}
	if got := detectHostOllama(ctx, saved); got != nil {
		t.Fatalf("already saved still suggested: %+v", got)
	}
	// Not answering means no suggestion.
	stubHostDetection(t, true, false, "172.27.224.1", false)
	if got := detectHostOllama(ctx, &trustantConfig{}); got != nil {
		t.Fatalf("unreachable host suggested: %+v", got)
	}
	// No default gateway under WSL.
	stubHostDetection(t, true, false, "", true)
	if got := detectHostOllama(ctx, &trustantConfig{}); got != nil {
		t.Fatalf("no gateway suggested: %+v", got)
	}
	// Lima.
	stubHostDetection(t, false, true, "", true)
	got = detectHostOllama(ctx, &trustantConfig{})
	if got == nil || got.URL != "http://host.lima.internal:11434" || got.Source != "Lima host" {
		t.Fatalf("Lima = %+v", got)
	}
	// Plain Linux: the local Ollama is the default, nothing to suggest.
	stubHostDetection(t, false, false, "192.168.1.1", true)
	if got := detectHostOllama(ctx, &trustantConfig{}); got != nil {
		t.Fatalf("plain Linux suggested: %+v", got)
	}
}

func TestOllamaInstancesListIncludesTheDetectedHost(t *testing.T) {
	withPredefinedEnvWorkspace(t, `{"provider":"ollama","base_url":"http://localhost:11434/v1","api_key":"dummy"}`)
	stubHostDetection(t, true, false, "172.27.224.1", true)

	_, st := ollamaInstancesCall(t, http.MethodGet, "")
	if st.Detected == nil || st.Detected.URL != "http://172.27.224.1:11434" {
		t.Fatalf("detected = %+v", st.Detected)
	}
	if len(st.Instances) != 0 || st.Active != "" {
		t.Fatalf("detection must not save or select anything: %+v", st)
	}

	// Adding it as a preset, like the button does, leaves it unselected and hides the suggestion.
	_, st = ollamaInstancesCall(t, http.MethodPost, `{"action":"save","instance":{"name":"Ollama host","url":"http://172.27.224.1:11434"}}`)
	if len(st.Instances) != 1 || st.Active != "" || st.Detected != nil {
		t.Fatalf("after add = %+v", st)
	}
	_, st = ollamaInstancesCall(t, http.MethodGet, "")
	if st.Detected != nil {
		t.Fatalf("saved host still suggested: %+v", st.Detected)
	}
}
