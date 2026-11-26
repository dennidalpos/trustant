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
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

// --- pruneSharedPool ------------------------------------------------------

func TestPruneSharedPoolRemovesBothPoolsForOneApp(t *testing.T) {
	cfg := &trustantConfig{
		PredefinedEnv: map[string]string{
			"APPSUITE__DB": "postgres://appsuite",
			"APPSUITE__S3": "s3://appsuite",
			"BILLING__DB":  "postgres://billing",
			"MY_API_KEY":   "typed by the user",
			"NO_SEPARATOR": "also the user's",
		},
		SharedProduction: map[string]map[string]map[string]string{
			"api.nuvolaris.io":   {"appsuite": {"APPSUITE__DB": "prod"}, "billing": {"BILLING__DB": "prod"}},
			"openserverless.dev": {"appsuite": {"APPSUITE__DB": "prod"}},
		},
	}

	removed := pruneSharedPool(cfg, "appsuite")

	// Two development entries and two production entries.
	if removed != 4 {
		t.Fatalf("removed = %d, want 4", removed)
	}
	for _, gone := range []string{"APPSUITE__DB", "APPSUITE__S3"} {
		if _, ok := cfg.PredefinedEnv[gone]; ok {
			t.Fatalf("%s survived the prune", gone)
		}
	}
	// Another app's exports are none of this app's business.
	if cfg.PredefinedEnv["BILLING__DB"] != "postgres://billing" {
		t.Fatalf("another app's export was pruned")
	}
	// Hand-typed entries are the user's, separator or not.
	if cfg.PredefinedEnv["MY_API_KEY"] == "" || cfg.PredefinedEnv["NO_SEPARATOR"] == "" {
		t.Fatalf("a hand-typed entry was pruned: %v", cfg.PredefinedEnv)
	}
	if _, ok := cfg.SharedProduction["api.nuvolaris.io"]["appsuite"]; ok {
		t.Fatalf("production entry survived the prune")
	}
	if cfg.SharedProduction["api.nuvolaris.io"]["billing"]["BILLING__DB"] != "prod" {
		t.Fatalf("another app's production export was pruned")
	}
	// A host whose last entry went with the app leaves no empty map behind:
	// the workspace file uses omitempty to stay small.
	if _, ok := cfg.SharedProduction["openserverless.dev"]; ok {
		t.Fatalf("emptied host map was kept: %v", cfg.SharedProduction)
	}
}

func TestPruneSharedPoolIsCaseInsensitiveOnTheAppName(t *testing.T) {
	// The pool name is uppercased by sharedVarPrefix; the app is stored as the
	// user typed it. Matching must fold case or nothing would ever be pruned.
	cfg := &trustantConfig{PredefinedEnv: map[string]string{"MyApp__DB": "v", "MYAPP__S3": "v"}}
	if removed := pruneSharedPool(cfg, "myapp"); removed != 2 {
		t.Fatalf("removed = %d, want 2 (%v)", removed, cfg.PredefinedEnv)
	}
}

func TestPruneSharedPoolDoesNotMatchAPrefixOfAnotherApp(t *testing.T) {
	// splitSharedVarName splits on the FIRST separator, so the prefix here is
	// BILLING. Deleting an app called `bill` must not take it with it.
	cfg := &trustantConfig{PredefinedEnv: map[string]string{
		"BILLING__POSTGRESDB": "postgres://billing",
		"BILL__POSTGRESDB":    "postgres://bill",
	}}

	if removed := pruneSharedPool(cfg, "bill"); removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
	if cfg.PredefinedEnv["BILLING__POSTGRESDB"] != "postgres://billing" {
		t.Fatalf("a different app's export was pruned by prefix collision: %v", cfg.PredefinedEnv)
	}
}

func TestPruneSharedPoolNilsAnEmptiedPool(t *testing.T) {
	cfg := &trustantConfig{
		PredefinedEnv:    map[string]string{"APPSUITE__DB": "v"},
		SharedProduction: map[string]map[string]map[string]string{"h": {"appsuite": {"APPSUITE__DB": "v"}}},
	}
	pruneSharedPool(cfg, "appsuite")
	if cfg.PredefinedEnv != nil || cfg.SharedProduction != nil {
		t.Fatalf("emptied pools were not nilled: %v %v", cfg.PredefinedEnv, cfg.SharedProduction)
	}
}

func TestPruneSharedPoolIsANoOpForAnAppThatSharedNothing(t *testing.T) {
	cfg := &trustantConfig{PredefinedEnv: map[string]string{"APPSUITE__DB": "v"}}
	if removed := pruneSharedPool(cfg, "somethingelse"); removed != 0 {
		t.Fatalf("removed = %d, want 0", removed)
	}
	if len(cfg.PredefinedEnv) != 1 {
		t.Fatalf("the pool was touched: %v", cfg.PredefinedEnv)
	}
	// An empty name must never be read as "prune everything".
	if removed := pruneSharedPool(cfg, "   "); removed != 0 || len(cfg.PredefinedEnv) != 1 {
		t.Fatalf("a blank app name pruned something: %d %v", removed, cfg.PredefinedEnv)
	}
}

// --- removeSharedPoolVar --------------------------------------------------

func TestRemoveSharedPoolVarSpansEveryPool(t *testing.T) {
	cfg := &trustantConfig{
		PredefinedEnv: map[string]string{"APPSUITE__DB": "v", "OTHER": "v"},
		SharedProduction: map[string]map[string]map[string]string{
			"api.nuvolaris.io":   {"appsuite": {"APPSUITE__DB": "v"}, "other": {"OTHER": "v"}},
			"openserverless.dev": {"appsuite": {"APPSUITE__DB": "v"}},
		},
	}

	// One development entry plus two production entries.
	if removed := removeSharedPoolVar(cfg, "APPSUITE__DB"); removed != 3 {
		t.Fatalf("removed = %d, want 3", removed)
	}
	if _, ok := cfg.PredefinedEnv["APPSUITE__DB"]; ok {
		t.Fatalf("development entry survived")
	}
	if _, ok := cfg.SharedProduction["openserverless.dev"]; ok {
		t.Fatalf("emptied host map was kept")
	}
	if cfg.SharedProduction["api.nuvolaris.io"]["other"]["OTHER"] != "v" {
		t.Fatalf("an unrelated entry was removed")
	}
}

func TestRemoveSharedPoolVarRemovesAHandTypedName(t *testing.T) {
	// The endpoint does not care who produced it; that distinction only governs
	// whether the removal is durable.
	cfg := &trustantConfig{PredefinedEnv: map[string]string{"MY_API_KEY": "v"}}
	if removed := removeSharedPoolVar(cfg, "MY_API_KEY"); removed != 1 {
		t.Fatalf("removed = %d, want 1", removed)
	}
}

// --- DELETE /api/predefined-env -------------------------------------------

func TestDeletePredefinedEnvEndpoint(t *testing.T) {
	isolateSharedWorkspace(t)
	saveWorkspaceConfig(&trustantConfig{
		Apps:          map[string]*AppConfig{"appsuite": {Password: "p"}},
		PredefinedEnv: map[string]string{"APPSUITE__DB": "secret", "MY_API_KEY": "v"},
		SharedProduction: map[string]map[string]map[string]string{
			"api.nuvolaris.io": {"appsuite": {"APPSUITE__DB": "prodsecret"}},
		},
	})

	rec := httptest.NewRecorder()
	handleDeletePredefinedEnv(rec, httptest.NewRequest("DELETE", "/api/predefined-env?name=APPSUITE__DB", nil))
	if rec.Code != 200 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}
	var got struct {
		Removed int `json:"removed"`
	}
	json.Unmarshal(rec.Body.Bytes(), &got)
	if got.Removed != 2 {
		t.Fatalf("removed = %d, want 2", got.Removed)
	}

	// It is the saved config that matters, not the response.
	wsCfg, err := loadWorkspaceConfig()
	if err != nil {
		t.Fatalf("reload: %s", err)
	}
	if _, ok := wsCfg.PredefinedEnv["APPSUITE__DB"]; ok {
		t.Fatalf("app-produced entry survived the delete")
	}
	if wsCfg.PredefinedEnv["MY_API_KEY"] != "v" {
		t.Fatalf("an unrelated entry was lost")
	}
	if _, ok := wsCfg.SharedProduction["api.nuvolaris.io"]; ok {
		t.Fatalf("production entry survived the delete")
	}
}

func TestDeletePredefinedEnvIsIdempotentAndNeedsAName(t *testing.T) {
	isolateSharedWorkspace(t)
	saveWorkspaceConfig(&trustantConfig{PredefinedEnv: map[string]string{"KEEP": "v"}})

	// Removing a name that is not there is success: a double-click is not an
	// error.
	rec := httptest.NewRecorder()
	handleDeletePredefinedEnv(rec, httptest.NewRequest("DELETE", "/api/predefined-env?name=NOT_THERE", nil))
	if rec.Code != 200 {
		t.Fatalf("absent name: %d %s", rec.Code, rec.Body.String())
	}

	// A missing name must not be read as "remove everything".
	rec = httptest.NewRecorder()
	handleDeletePredefinedEnv(rec, httptest.NewRequest("DELETE", "/api/predefined-env", nil))
	if rec.Code != 400 {
		t.Fatalf("missing name: %d, want 400", rec.Code)
	}
	wsCfg, _ := loadWorkspaceConfig()
	if wsCfg.PredefinedEnv["KEEP"] != "v" {
		t.Fatalf("the pool was emptied by a nameless delete: %v", wsCfg.PredefinedEnv)
	}
}

// TestBulkSaveStillCarriesAppProducedEntries guards the asymmetry this feature
// rests on: the DELETE is the only way to drop an app-produced entry, and the
// bulk POST must not quietly become a second one. It receives the whole set, so
// honouring an absent key there would let a stale tab drop a live export.
func TestBulkSaveStillCarriesAppProducedEntries(t *testing.T) {
	isolateSharedWorkspace(t)
	saveWorkspaceConfig(&trustantConfig{
		Apps:          map[string]*AppConfig{"appsuite": {Password: "p"}},
		PredefinedEnv: map[string]string{"APPSUITE__DB": "secret", "MY_API_KEY": "v"},
	})

	// A save that omits the app-produced row entirely.
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/api/predefined-env",
		strings.NewReader(`{"vars":[{"name":"MY_API_KEY","value":"v"}]}`))
	handlePostPredefinedEnv(rec, req)
	if rec.Code != 200 {
		t.Fatalf("post: %d %s", rec.Code, rec.Body.String())
	}

	wsCfg, _ := loadWorkspaceConfig()
	if wsCfg.PredefinedEnv["APPSUITE__DB"] != "secret" {
		t.Fatalf("the bulk save dropped an app-produced entry: %v", wsCfg.PredefinedEnv)
	}
}

// --- deleting the app -----------------------------------------------------

func installDeleteUserFixture(t *testing.T, root string) string {
	t.Helper()
	const fixture = `#!/bin/sh
set -eu
printf '%s\n' "$*" >> "$TEST_DELETE_ROOT/ops.log"
case "$1 $2" in
  "util kubectl")
    [ "$3 $4 $6 $7 $8" = "-- get --ignore-not-found -o name" ]
    app=${5#wsku/}
    if [ -f "$TEST_DELETE_ROOT/$app.probe-once" ]; then
      rm -f -- "$TEST_DELETE_ROOT/$app.probe-once"
      exit 2
    fi
    if [ -f "$TEST_DELETE_ROOT/$app.user" ]; then printf 'whiskuser/%s\n' "$app"; fi
    ;;
  "admin deleteuser")
    if [ -f "$TEST_DELETE_ROOT/$3.deny" ]; then exit 2; fi
    if [ -f "$TEST_DELETE_ROOT/$3.delete-once" ]; then
      rm -f -- "$TEST_DELETE_ROOT/$3.delete-once"
      exit 2
    fi
    rm -f -- "$TEST_DELETE_ROOT/$3.user"
    if [ -f "$TEST_DELETE_ROOT/$3.removed-with-error" ]; then exit 2; fi
    ;;
  *) exit 3 ;;
esac
`
	if err := os.WriteFile(filepath.Join(root, "ops"), []byte(fixture), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", root+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("TEST_DELETE_ROOT", root)
	return filepath.Join(root, "ops.log")
}

func TestDeleteOpenServerlessUserRecoversFromFailedCommands(t *testing.T) {
	const app = "fixtureuser"
	for _, tc := range []struct {
		name        string
		failures    []string
		wantDeletes int
	}{
		{"transient check and deletion", []string{"probe-once", "delete-once"}, 2},
		{"user removed despite command error", []string{"removed-with-error"}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			opsLog := installDeleteUserFixture(t, root)
			for _, marker := range append([]string{"user"}, tc.failures...) {
				if err := os.WriteFile(filepath.Join(root, app+"."+marker), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := deleteOpenServerlessUser(context.Background(), app); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(filepath.Join(root, app+".user")); !os.IsNotExist(err) {
				t.Fatalf("fixture user should be absent: %v", err)
			}
			commands, err := os.ReadFile(opsLog)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(string(commands), "admin deleteuser "+app+"\n"); got != tc.wantDeletes {
				t.Fatalf("want %d deletion attempts, got %d: %s", tc.wantDeletes, got, commands)
			}
		})
	}
}

func TestDeleteRepoPreservesOtherRuntimeAndCanRetryPartialFailures(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node is required for the TruACP fixture")
	}
	root := isolateSharedWorkspace(t)
	opsLog := installDeleteUserFixture(t, root)
	const active, inactive = "fixtureactive", "fixtureinactive"
	workspaceApp := filepath.Join(WorkspaceDir, "workspace", inactive)
	for _, app := range []string{active, inactive} {
		makeWorkbench(t, app)
		if err := os.MkdirAll(filepath.Join(WorkspaceDir, "workspace", app), 0755); err != nil {
			t.Fatal(err)
		}
	}
	if err := saveWorkspaceConfig(&trustantConfig{
		Apps: map[string]*AppConfig{active: {Password: "active-password"}, inactive: {Password: "inactive-password"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeCurrentApp(active); err != nil {
		t.Fatal(err)
	}
	activePath, err := canonicalWorkbenchPath(active)
	if err != nil {
		t.Fatal(err)
	}
	leaderScript := filepath.Join(root, "truacp.cjs")
	if err := os.WriteFile(leaderScript, []byte("setInterval(() => {}, 1000);\n"), 0600); err != nil {
		t.Fatal(err)
	}
	leader := exec.Command(node, leaderScript)
	leader.Dir = activePath
	leader.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := leader.Start(); err != nil {
		t.Fatal(err)
	}
	pgid := leader.Process.Pid
	leaderExited := make(chan struct{})
	go func() {
		_ = leader.Wait()
		close(leaderExited)
	}()
	t.Cleanup(func() {
		select {
		case <-leaderExited:
		default:
			_ = leader.Process.Kill()
			<-leaderExited
		}
	})
	if err := writePgid(pgid); err != nil {
		t.Fatal(err)
	}

	deleteApp := func(app string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		handleRepo(rec, httptest.NewRequest(http.MethodDelete, "/api/repo",
			strings.NewReader(`{"name":"`+app+`"}`)))
		return rec
	}
	writeMarker := func(path, value string) {
		t.Helper()
		if err := os.WriteFile(path, []byte(value), 0600); err != nil {
			t.Fatal(err)
		}
	}
	assertRuntimeIntact := func() {
		t.Helper()
		if current, err := readPgid(); err != nil || current != pgid || !managedRuntimeOwnsGroup(active, activePath, pgid) {
			t.Fatalf("the other application's runtime changed: pgid=%d err=%v", current, err)
		}
	}
	assertExists := func(path string) {
		t.Helper()
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	remoteUser := filepath.Join(root, inactive+".user")
	denied := filepath.Join(root, inactive+".deny")
	writeMarker(remoteUser, "")
	writeMarker(denied, "")

	// A launch or stop in progress uses the same lifecycle lock.
	release := lockRuntimeLifecycle("launch fixture")
	result := make(chan *httptest.ResponseRecorder, 1)
	started := make(chan struct{})
	go func() {
		close(started)
		result <- deleteApp(inactive)
	}()
	<-started
	select {
	case rec := <-result:
		release()
		t.Fatalf("delete ran during another lifecycle operation: %d %s", rec.Code, rec.Body)
	case <-time.After(50 * time.Millisecond):
		release()
	}
	rec := <-result
	if rec.Code != http.StatusBadGateway || !strings.Contains(rec.Body.String(), "available for retry") {
		t.Fatalf("failed remote deletion returned %d: %s", rec.Code, rec.Body)
	}
	assertRuntimeIntact()
	assertExists(workspaceApp)
	assertExists(filepath.Join(WorkbenchDir, inactive))
	assertExists(remoteUser)

	// A local failure after remote success must keep the app available to retry.
	if err := os.Remove(denied); err != nil {
		t.Fatal(err)
	}
	cfg, err := loadWorkspaceConfig()
	if err != nil || cfg.Apps[inactive] == nil {
		t.Fatalf("app configuration was not retained: %v", err)
	}
	writeMarker(filepath.Join(WorkspaceDir, "trustant.json"), "{")
	rec = deleteApp(inactive)
	if rec.Code != http.StatusInternalServerError || !strings.Contains(rec.Body.String(), "Configuration cleanup failed") {
		t.Fatalf("partial deletion returned %d: %s", rec.Code, rec.Body)
	}
	assertRuntimeIntact()
	assertExists(workspaceApp)
	if _, err := os.Stat(remoteUser); !os.IsNotExist(err) {
		t.Fatalf("fixture user should already be absent: %v", err)
	}
	if err := saveWorkspaceConfig(cfg); err != nil {
		t.Fatal(err)
	}
	if rec = deleteApp(inactive); rec.Code != http.StatusNoContent {
		t.Fatalf("retry returned %d: %s", rec.Code, rec.Body)
	}
	assertRuntimeIntact()
	if _, err := os.Stat(workspaceApp); !os.IsNotExist(err) {
		t.Fatalf("retry did not remove the repository: %v", err)
	}
	commands, err := os.ReadFile(opsLog)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(commands), "admin deleteuser "+inactive+"\n"); got != 4 {
		t.Fatalf("want three denied attempts and one successful user deletion, got %d: %s", got, commands)
	}

	if rec = deleteApp(active); rec.Code != http.StatusNoContent {
		t.Fatalf("active app deletion returned %d: %s", rec.Code, rec.Body)
	}
	select {
	case <-leaderExited:
	case <-time.After(time.Second):
		t.Fatal("deleted application's runtime did not stop")
	}
	if _, err := readPgid(); !os.IsNotExist(err) {
		t.Fatalf("active app deletion retained the PGID marker: %v", err)
	}
	if _, err := readCurrentApp(); !os.IsNotExist(err) {
		t.Fatalf("active app deletion retained the current marker: %v", err)
	}
}

// TestDeleteRepoPrunesWhatTheAppShared is the whole point of the issue: the
// pool is append-only, so without the prune a deleted app's resolved secrets
// outlive it.
func TestDeleteRepoPrunesWhatTheAppShared(t *testing.T) {
	root := isolateSharedWorkspace(t)
	installDeleteUserFixture(t, root)
	// handleDeleteRepo requires the bare repo to exist before it will act.
	if err := os.MkdirAll(filepath.Join(WorkspaceDir, "workspace", "appsuite"), 0755); err != nil {
		t.Fatalf("mkdir bare repo: %s", err)
	}
	saveWorkspaceConfig(&trustantConfig{
		Apps: map[string]*AppConfig{
			"appsuite": {Password: "p"},
			"billing":  {Password: "p"},
		},
		PredefinedEnv: map[string]string{
			"APPSUITE__DB": "postgres://user:pass@appsuite",
			"BILLING__DB":  "postgres://billing",
			"MY_API_KEY":   "typed by the user",
		},
		SharedProduction: map[string]map[string]map[string]string{
			"api.nuvolaris.io": {"appsuite": {"APPSUITE__DB": "prodsecret"}, "billing": {"BILLING__DB": "prod"}},
		},
	})

	rec := httptest.NewRecorder()
	handleDeleteRepo(rec, httptest.NewRequest("DELETE", "/api/repo",
		strings.NewReader(`{"name":"appsuite"}`)))
	if rec.Code != 204 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}

	wsCfg, err := loadWorkspaceConfig()
	if err != nil {
		t.Fatalf("reload: %s", err)
	}
	if _, ok := wsCfg.Apps["appsuite"]; ok {
		t.Fatalf("the app entry survived")
	}
	if _, ok := wsCfg.PredefinedEnv["APPSUITE__DB"]; ok {
		t.Fatalf("a deleted app's secret is still in the development pool: %v", wsCfg.PredefinedEnv)
	}
	if _, ok := wsCfg.SharedProduction["api.nuvolaris.io"]["appsuite"]; ok {
		t.Fatalf("a deleted app's secret is still in the production pool")
	}
	// Everything else is untouched, in the same single save.
	if wsCfg.PredefinedEnv["BILLING__DB"] == "" || wsCfg.PredefinedEnv["MY_API_KEY"] == "" {
		t.Fatalf("the prune took more than the app's own: %v", wsCfg.PredefinedEnv)
	}
	if _, ok := wsCfg.Apps["billing"]; !ok {
		t.Fatalf("another app was removed")
	}
}

// TestRecreatingADeletedAppDoesNotInheritItsSecrets is the sharpest
// consequence of the old behaviour. Ownership is derived from the name prefix
// matching an app that exists, so orphaned entries were reclaimed as
// app-produced the moment a same-named app reappeared — handing a brand-new
// app the deleted app's credentials until the first refresh.
func TestRecreatingADeletedAppDoesNotInheritItsSecrets(t *testing.T) {
	root := isolateSharedWorkspace(t)
	installDeleteUserFixture(t, root)
	if err := os.MkdirAll(filepath.Join(WorkspaceDir, "workspace", "appsuite"), 0755); err != nil {
		t.Fatalf("mkdir bare repo: %s", err)
	}
	saveWorkspaceConfig(&trustantConfig{
		Apps:          map[string]*AppConfig{"appsuite": {Password: "p"}},
		PredefinedEnv: map[string]string{"APPSUITE__DB": "the old app's database"},
	})

	rec := httptest.NewRecorder()
	handleDeleteRepo(rec, httptest.NewRequest("DELETE", "/api/repo", strings.NewReader(`{"name":"appsuite"}`)))
	if rec.Code != 204 {
		t.Fatalf("delete: %d %s", rec.Code, rec.Body.String())
	}

	// A new, unrelated app of the same name.
	wsCfg, _ := loadWorkspaceConfig()
	if wsCfg.Apps == nil {
		wsCfg.Apps = map[string]*AppConfig{}
	}
	wsCfg.Apps["appsuite"] = &AppConfig{Password: "p2"}
	saveWorkspaceConfig(wsCfg)

	wsCfg, _ = loadWorkspaceConfig()
	if v, ok := wsCfg.PredefinedEnv["APPSUITE__DB"]; ok {
		t.Fatalf("the new app inherited the deleted app's credential: %q", v)
	}
}

// TestConsumerOfAPrunedProducerRePends closes the loop. resolveOneImport
// already re-pends a reference whose target left the pool rather than falling
// back to a different producer; until the prune existed that branch could
// never fire, because nothing ever left.
func TestConsumerOfAPrunedProducerRePends(t *testing.T) {
	cfg := &trustantConfig{PredefinedEnv: map[string]string{
		"APPSUITE__DB": "postgres://appsuite",
		"BILLING__DB":  "postgres://billing",
	}}
	// The consumer is bound to appsuite by reference.
	dev := map[string]string{"EXT_URL": "${{APPSUITE__DB}}"}
	binding := EnvBinding{Name: "EXT_URL", Pattern: "*__DB"}

	if res := resolveOneImport(binding, dev, cfg.PredefinedEnv); res.Pending {
		t.Fatalf("resolved import was pending before the prune")
	}

	pruneSharedPool(cfg, "appsuite")

	res := resolveOneImport(binding, dev, cfg.PredefinedEnv)
	if !res.Pending {
		t.Fatalf("a consumer of a pruned producer did not re-pend: %+v", res)
	}
	// Silently falling back to BILLING__DB would point the app at a different
	// database without telling anyone.
	if res.Value == "postgres://billing" {
		t.Fatalf("the import silently fell back to another producer")
	}
}
