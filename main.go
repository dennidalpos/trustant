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
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed web
var embeddedWeb embed.FS

//go:embed _build.txt
var buildTxt string

//go:embed openserverless-instructions.md
var openserverlessInstructionsMd string

//go:embed openserverless-contract.md
var openserverlessContractMd string

//go:embed app-agents.md
var appAgentsMd string

// The checker is embedded rather than copied from the host. WHY: every app
// must receive the exact source-contract checker matching this Trustant
// binary, including its managed-live behavior that does not poll deploy ZIPs.
//
//go:embed check_openserverless_actions.sh
var openserverlessCheckerSh string

//go:embed check_trustant_frontend.sh
var trustantFrontendCheckerSh string

//go:embed check_trustant_app.sh
var trustantAppCheckerSh string

//go:embed milvus_cli.tmpl
var milvusCliTemplate string

// noStoreHTML prevents an upgraded Trustant UI from restoring stale inline
// configuration logic from browser history. This is especially important for
// hard schema cutovers such as OpenCode -> Pi, where old JavaScript can create
// a redirect loop even though the server-side configuration is already valid.
//
// Stylesheets and scripts are served with no-cache: the browser may keep them
// but must revalidate (a cheap 304 via Last-Modified) on every load. Without a
// Cache-Control header the browser applies heuristic freshness and can pair a
// new HTML page with a stale CSS or JS file for days after an upgrade.
// Request body limits. Ordinary API calls carry small JSON; /api/upload takes
// user files and gets a larger, still bounded, allowance.
const (
	maxRequestBody int64 = 8 << 20
	maxUploadBody  int64 = 512 << 20
)

// limitBody caps every request body so no handler can be made to buffer or
// spool an unbounded amount. A declared Content-Length over the limit is
// rejected up front with 413; undeclared (chunked) bodies are cut off by
// http.MaxBytesReader and surface as a read error in the handler.
func limitBody(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		limit := maxRequestBody
		if r.URL.Path == "/api/upload" {
			limit = maxUploadBody
		}
		if r.ContentLength > limit {
			http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
			return
		}
		if r.Body != nil {
			r.Body = http.MaxBytesReader(w, r.Body, limit)
		}
		next.ServeHTTP(w, r)
	})
}

func noStoreHTML(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/" || strings.HasSuffix(r.URL.Path, ".html"):
			w.Header().Set("Cache-Control", "no-store")
		case strings.HasSuffix(r.URL.Path, ".css") || strings.HasSuffix(r.URL.Path, ".js"):
			w.Header().Set("Cache-Control", "no-cache")
		}
		next.ServeHTTP(w, r)
	})
}

// authWarning describes the exposure when local authentication is off. The
// server always listens on every interface, so a disabled manager means anyone
// who can reach the port gets the terminal, file and deploy APIs.
func authWarning(a *authManager) string {
	if a != nil && a.enabled {
		return ""
	}
	return "WARNING: authentication is disabled and Trustant listens on all interfaces on port 8910 over plain HTTP. " +
		"Keep it on a trusted network or set TRUSTANT_AUTH_MODE=local (see spec/authentication.md)."
}

func main() {
	auth, err := newAuthManagerFromEnv()
	if err != nil {
		log.Fatalf("Authentication configuration failed: %s", err)
	}

	// Run preflight checks
	if err := runPreflight(); err != nil {
		log.Fatalf("Preflight checks failed: %s", err)
	}

	// Ensure workspace and workbench directories exist
	wsPath := filepath.Join(WorkspaceDir, "workspace")
	if err := os.MkdirAll(wsPath, 0755); err != nil {
		log.Printf("Warning: failed to create workspace directory: %s", err)
	}
	if err := os.MkdirAll(WorkbenchDir, 0755); err != nil {
		log.Printf("Warning: failed to create workbench directory: %s", err)
	}

	// Terminate any leftover processes from previous run (kept for backward compatibility)
	terminateLeftoverProcesses()

	// Recreate missing ephemeral checkouts from durable bare repos. TruACP/Pi is
	// launched with those paths as cwd, so restoring them before serving requests
	// keeps persisted applications editable across pod rebuilds and VM restarts.
	restoreMissingWorkbenchCheckouts()

	// Parse version info
	parseVersion(buildTxt)

	// Probe the ops CLI once. It shells out, so it must not run per request;
	// /api/version serves the cached result. Non-fatal if ops is absent.
	probeOpsInfo()

	// API routes
	http.HandleFunc("/api/version", handleVersion)
	http.HandleFunc("/api/initstatus", handleInitStatus)
	http.HandleFunc("/api/status", handleStatus)
	http.HandleFunc("/api/repo", handleRepo)
	http.HandleFunc("/api/starters", handleStarters)
	http.HandleFunc("/api/upload", handleUpload)
	http.HandleFunc("/api/launch/", handleLaunch)
	http.HandleFunc("/api/launch", handleLaunch)
	http.HandleFunc("/api/git", handleGit)
	http.HandleFunc("/api/git/status/", handleGitStatus)
	http.HandleFunc("/api/git/save", handleGitSave)
	http.HandleFunc("/api/git/pull", handleGitPull)
	http.HandleFunc("/api/git/deploy", handleGitDeploy)
	http.HandleFunc("/api/github/status", handleGitHubStatus)
	http.HandleFunc("/api/github/login", handleGitHubLogin)
	http.HandleFunc("/api/github/login/cancel", handleGitHubLoginCancel)
	http.HandleFunc("/api/github/logout", handleGitHubLogout)
	http.HandleFunc("/api/github/repos", handleGitHubRepos)
	http.HandleFunc("/api/publish/force-push", handlePublish)
	http.HandleFunc("/api/sshkey", handleSSHKey)
	http.HandleFunc("/api/configure", handleConfigure)
	http.HandleFunc("/api/testmodel", handleTestModel)
	http.HandleFunc("/api/ollama-connect", handleOllamaConnect)
	http.HandleFunc("/api/discover-models", handleDiscoverModels)
	http.HandleFunc("/api/configuration", handleConfiguration)
	http.HandleFunc("/api/predefined-env", handlePredefinedEnv)
	http.HandleFunc("/api/imports/", handleImports)
	http.HandleFunc("/api/shared", handleShared)
	http.HandleFunc("/api/shared/", handleShared)
	http.HandleFunc("/api/appconfig/", handleAppConfig)
	http.HandleFunc("/api/publish/push", handlePublish)
	http.HandleFunc("/api/publish/remote", handlePublish)
	http.HandleFunc("/api/publish/provision", handlePublish)
	http.HandleFunc("/api/skills/", handleSkills)
	http.HandleFunc("/api/memory/", handleMemory)
	http.HandleFunc("/api/terminal/", handleTerminal)
	http.HandleFunc("/api/files/", handleFiles)
	http.HandleFunc("/api/redeploy", handleRedeploy)
	http.HandleFunc("/api/undeploy", handleUndeploy)
	http.HandleFunc("/api/clean", handleClean)
	http.HandleFunc("/api/activations/poll", handleActivationPoll)
	http.HandleFunc("/api/license", handleLicense)
	http.HandleFunc("/api/license/settings", handleLicenseSettings)
	http.HandleFunc("/api/legal/", handleLegal)
	http.HandleFunc("/api/credits", handleCredits)
	http.HandleFunc("/api/topup", handleTopUp)
	auth.registerRoutes(http.DefaultServeMux)

	// Static file serving
	if _, err := os.Stat("web"); err == nil {
		// Serve from disk (development mode)
		log.Println("Serving from disk: ./web")
		http.Handle("/", noStoreHTML(http.FileServer(http.Dir("web"))))
	} else {
		// Serve from embedded filesystem
		log.Println("Serving from embedded filesystem")
		webFS, err := fs.Sub(embeddedWeb, "web")
		if err != nil {
			log.Fatal(err)
		}
		http.Handle("/", noStoreHTML(http.FileServer(http.FS(webFS))))
	}

	if warning := authWarning(auth); warning != "" {
		log.Println(warning)
	}
	log.Println("Starting server on :8910")
	// Wrap with hostname verification middleware (implements spec points 7 & 8)
	handler := hostnameMiddleware(auth.middleware(limitBody(http.DefaultServeMux)))
	if err := http.ListenAndServe(":8910", handler); err != nil {
		log.Fatal(err)
	}
}
