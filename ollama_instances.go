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
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Named Ollama servers the "ollama" provider can be pointed at, for example
// different clusters. The default (no instance selected) is the Ollama of this
// machine, OLLAMA_ENDPOINT, exactly as before. See spec/2a-config.md
// "Ollama instances".

const (
	maxOllamaInstances = 50
	localOllamaV1URL   = "http://localhost:11434/v1"
)

// ollamaInstance is a saved Ollama server. URL is the root without /v1. The
// API key is not part of it: it is kept in stateDir/secrets, so it never
// reaches trustant.json or the browser.
type ollamaInstance struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	URL  string `json:"url"`
}

var ollamaInstanceIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

func findOllamaInstance(cfg *trustantConfig, id string) *ollamaInstance {
	if cfg == nil || id == "" {
		return nil
	}
	for i := range cfg.OllamaInstances {
		if cfg.OllamaInstances[i].ID == id {
			return &cfg.OllamaInstances[i]
		}
	}
	return nil
}

func ollamaInstanceKeyPath(id string) string {
	return stateDir("secrets", "ollama-instances", id)
}

// readOllamaInstanceKey returns the stored key of an instance, or "".
func readOllamaInstanceKey(id string) string {
	if !ollamaInstanceIDPattern.MatchString(id) {
		return ""
	}
	data, err := os.ReadFile(ollamaInstanceKeyPath(id))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func writeOllamaInstanceKey(id, key string) error {
	path := ollamaInstanceKeyPath(id)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(key+"\n"), 0o600)
}

func removeOllamaInstanceKey(id string) {
	if ollamaInstanceIDPattern.MatchString(id) {
		_ = os.Remove(ollamaInstanceKeyPath(id))
	}
}

// effectiveAPIKey is the key to send to the configured provider: the stored key
// of the selected Ollama instance, otherwise cfg.APIKey.
func effectiveAPIKey(cfg *trustantConfig) string {
	if cfg.Provider == "ollama" {
		if inst := findOllamaInstance(cfg, cfg.OllamaActive); inst != nil {
			return readOllamaInstanceKey(inst.ID)
		}
	}
	return strings.TrimSpace(cfg.APIKey)
}

// normalizeOllamaInstanceURL accepts http(s)://host[:port] with an optional
// /v1 and returns the root without it.
func normalizeOllamaInstanceURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	parsed, err := url.Parse(raw)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("the URL must look like http://host:11434 (optionally ending in /v1)")
	}
	switch strings.TrimRight(parsed.Path, "/") {
	case "", "/v1":
	default:
		return "", fmt.Errorf("the URL must look like http://host:11434 (optionally ending in /v1)")
	}
	return parsed.Scheme + "://" + parsed.Host, nil
}

func slugifyOllamaInstanceName(name string) string {
	var b strings.Builder
	dash := false
	for _, r := range strings.ToLower(name) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			dash = false
		default:
			if b.Len() > 0 && !dash {
				b.WriteByte('-')
				dash = true
			}
		}
	}
	slug := strings.Trim(b.String(), "-")
	if len(slug) > 32 {
		slug = strings.Trim(slug[:32], "-")
	}
	if slug == "" {
		slug = "ollama"
	}
	return slug
}

func uniqueOllamaInstanceID(cfg *trustantConfig, name string) string {
	base := slugifyOllamaInstanceName(name)
	id := base
	for n := 2; findOllamaInstance(cfg, id) != nil; n++ {
		id = fmt.Sprintf("%s-%d", base, n)
	}
	return id
}

type ollamaInstanceView struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	URL    string `json:"url"`
	KeySet bool   `json:"key_set"`
}

type ollamaInstancesState struct {
	Instances []ollamaInstanceView `json:"instances"`
	Active    string               `json:"active"`
	// BaseURL is the provider base_url after the change, so the page can keep
	// its in-memory configuration in step.
	BaseURL string `json:"base_url,omitempty"`
	Error   string `json:"error,omitempty"`
}

func ollamaInstancesView(cfg *trustantConfig) ollamaInstancesState {
	state := ollamaInstancesState{Instances: []ollamaInstanceView{}, Active: cfg.OllamaActive, BaseURL: cfg.BaseURL}
	for _, inst := range cfg.OllamaInstances {
		state.Instances = append(state.Instances, ollamaInstanceView{
			ID: inst.ID, Name: inst.Name, URL: inst.URL, KeySet: readOllamaInstanceKey(inst.ID) != "",
		})
	}
	return state
}

// applyOllamaSelection points the provider at the selected instance ("" =
// default Ollama). The saved base_url is only touched while the "ollama"
// provider is in use, so a Private AI endpoint is never overwritten.
func applyOllamaSelection(cfg *trustantConfig, id string) {
	cfg.OllamaActive = id
	if cfg.Provider != "ollama" {
		return
	}
	cfg.APIKey = "dummy"
	if inst := findOllamaInstance(cfg, id); inst != nil {
		cfg.BaseURL = inst.URL + "/v1"
	} else {
		cfg.OllamaActive = ""
		cfg.BaseURL = localOllamaV1URL
	}
}

type ollamaInstanceRequest struct {
	Action   string `json:"action"`
	ID       string `json:"id"`
	Instance struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		URL      string `json:"url"`
		APIKey   string `json:"api_key"`
		ClearKey bool   `json:"clear_key"`
	} `json:"instance"`
}

type ollamaInstanceError struct {
	status int
	msg    string
}

func (e *ollamaInstanceError) Error() string { return e.msg }

func writeOllamaInstancesJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// handleOllamaInstances lists (GET) and edits (POST) the saved Ollama
// instances and the selected one.
func handleOllamaInstances(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		cfg, err := loadTrustantConfig()
		if err != nil {
			writeOllamaInstancesJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeOllamaInstancesJSON(w, http.StatusOK, ollamaInstancesView(cfg))
	case http.MethodPost:
		var req ollamaInstanceRequest
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<14)).Decode(&req); err != nil {
			writeOllamaInstancesJSON(w, http.StatusBadRequest, map[string]string{"error": "Invalid JSON: " + err.Error()})
			return
		}
		var result *trustantConfig
		err := updateWorkspaceConfig(func(cfg *trustantConfig) error {
			if err := applyOllamaInstanceRequest(cfg, req); err != nil {
				return err
			}
			result = cfg
			return nil
		})
		if err != nil {
			status := http.StatusInternalServerError
			if ie, ok := err.(*ollamaInstanceError); ok {
				status = ie.status
			}
			writeOllamaInstancesJSON(w, status, map[string]string{"error": err.Error()})
			return
		}
		merged, mergeErr := loadTrustantConfig()
		if mergeErr != nil {
			merged = result
		}
		writeOllamaInstancesJSON(w, http.StatusOK, ollamaInstancesView(merged))
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// applyOllamaInstanceRequest mutates the workspace config for one request. Key
// files are written after validation, so a rejected request leaves no trace.
func applyOllamaInstanceRequest(cfg *trustantConfig, req ollamaInstanceRequest) error {
	bad := func(msg string) error { return &ollamaInstanceError{http.StatusBadRequest, msg} }
	switch req.Action {
	case "save":
		name := strings.TrimSpace(req.Instance.Name)
		if name == "" || len(name) > 60 {
			return bad("Give the instance a name (up to 60 characters).")
		}
		root, err := normalizeOllamaInstanceURL(req.Instance.URL)
		if err != nil {
			return bad(err.Error())
		}
		id := strings.TrimSpace(req.Instance.ID)
		existing := findOllamaInstance(cfg, id)
		if id != "" && existing == nil {
			return &ollamaInstanceError{http.StatusNotFound, "Unknown Ollama instance"}
		}
		for _, other := range cfg.OllamaInstances {
			if other.ID != id && strings.EqualFold(other.Name, name) {
				return bad("An instance with that name already exists.")
			}
		}
		if existing == nil {
			if len(cfg.OllamaInstances) >= maxOllamaInstances {
				return bad("Too many instances.")
			}
			id = uniqueOllamaInstanceID(cfg, name)
			cfg.OllamaInstances = append(cfg.OllamaInstances, ollamaInstance{ID: id, Name: name, URL: root})
		} else {
			existing.Name, existing.URL = name, root
			if cfg.OllamaActive == id && cfg.Provider == "ollama" {
				cfg.BaseURL = root + "/v1"
			}
		}
		key := strings.TrimSpace(req.Instance.APIKey)
		switch {
		case key != "":
			if err := writeOllamaInstanceKey(id, key); err != nil {
				return err
			}
		case req.Instance.ClearKey:
			removeOllamaInstanceKey(id)
		}
		return nil
	case "delete":
		kept := cfg.OllamaInstances[:0:0]
		for _, inst := range cfg.OllamaInstances {
			if inst.ID != req.ID {
				kept = append(kept, inst)
			}
		}
		if len(kept) == len(cfg.OllamaInstances) {
			return nil // idempotent, like the other delete buttons
		}
		cfg.OllamaInstances = kept
		removeOllamaInstanceKey(req.ID)
		if cfg.OllamaActive == req.ID {
			applyOllamaSelection(cfg, "")
		}
		return nil
	case "select":
		if req.ID != "" && findOllamaInstance(cfg, req.ID) == nil {
			return &ollamaInstanceError{http.StatusNotFound, "Unknown Ollama instance"}
		}
		if cfg.Provider != "ollama" {
			return &ollamaInstanceError{http.StatusConflict, "Ollama is not the selected provider."}
		}
		applyOllamaSelection(cfg, req.ID)
		return nil
	}
	return bad(`action must be "save", "delete" or "select"`)
}
