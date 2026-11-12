# AGENTS.md

<!-- secondbrain:repository:begin -->
Direttive comuni applicate: revisione **1.1** — 2026-11-12. <!-- secondbrain:source global=8238B8C140EC9FE3C6E4002D94B0AADBB1EAE0E15A52566BF31854D2292AC06C workstation=CFB3267FB25931AF68B8791B8D01F31AC30D940679D6D3443567A45BE9FBA9A6 -->
## SecondBrain

Repository knowledge lives in the SecondBrain dossier; code, guides, and publishable assets stay here.

- Load the slim startup with the SecondBrain checkout's scripts/knowledge.ps1 -Startup or MCP get_startup_context, and follow its references for detail. If the root is unreachable, report the dossier as unavailable.
- Write only to this repository's dossier `01-projects/<repo>/`, only through SecondBrain scripts or MCP write tools. Work folders come from scripts/work.ps1 -Init; sessions, retention, and cleanup rules load from `sb prompt`.
- Commits, pushes, deletions, and publication require specific authorization.
<!-- secondbrain:repository:end -->

## Scope and authorization

- Work within the user's request. Audits and reviews are read-only. For substantial changes, make a short plan and use an existing issue when the user provides one.
- Create or edit GitHub issues, PRs, and wiki pages only when explicitly requested. Commits, pushes, merges, releases, deployments, and destructive operations each require explicit authorization for the relevant scope. Check what scripts do before running them.
- Preserve unrelated work and keep credentials in their authentication stores. Before publishing, verify the destination and scan the working tree, index, and relevant history for secrets and private data. Follow [CONTRIBUTING.md](CONTRIBUTING.md) for an authorized PR.
- Use [.gitmodules](.gitmodules) and follow the instructions inside each affected submodule. Update only the submodules and gitlinks required by the change. Push to plugin repositories only with explicit authorization for those repositories.

## Changes and verification

- Verify current behavior in code and tests. Use [spec/](spec/) for intended behavior and update the relevant spec when behavior changes. Flag conflicts between the spec and the implementation. For a new API, register its route in [main.go](main.go) and keep its handler with the feature.
- Use `./start.ps1` from PowerShell on Windows (WSL2) or `./start.sh` on macOS (Lima) and native Ubuntu. Run `./run.sh` in the Ubuntu environment. See [spec/start.md](spec/start.md) and [spec/run.md](spec/run.md). Use [.env.dist](.env.dist) and [spec/0-preflight.md](spec/0-preflight.md) for environment setup.
- Run relevant checks. The Go suite (`go test ./...`) requires the ignored `_build.txt` embedded by [main.go](main.go). The development loop generates it before starting Air. Node checks and E2E entry points are listed in [package.json](package.json). Cluster scenarios may launch apps or call model providers. See [spec/8-e2e.md](spec/8-e2e.md).
- Review the diff and `git status --ignored` before handing off. Report checks run, failures, and unverified behavior.

## Trustant conventions

- Use **Trustant** for the product and `github.com/trustant/trustant` for the canonical repository. Preserve the module path in [go.mod](go.mod) and the upstream identities in [.gitmodules](.gitmodules).
- The durable app repository is bare and lives at `$WORKSPACE_DIR/workspace/<name>/`. The active checkout lives at `$WORKBENCH_DIR/<name>/` and receives regenerated configuration on launch. See [spec/4-launch.md](spec/4-launch.md) and [spec/13-gitignore.md](spec/13-gitignore.md).
- Use an FQDN for checks that exercise [hostname routing](middleware.go). The current host labels are `trustant` for the UI/API, `truacp` for the assistant, and `vite` for the app preview. Registered app hosts route to their development services.
- Treat [openserverless-instructions.md](openserverless-instructions.md) and [app-agents.md](app-agents.md) as product assets that guide generated apps. Preserve the license error prefixes consumed by the UI when changing that API. See [spec/14-license.md](spec/14-license.md).

## Release script side effects

- [build.sh](build.sh) in `--tag` and `--build` modes replaces local Git tags, updates `oplugins-truinst/opsroot.json`, and attempts `git commit -a`. With `--build --no-deploy`, those effects still occur. The regular deploy reads the image from `opsroot.json`.
- [hotfix.sh](hotfix.sh) can also replace local tags. Its deploy patches the live StatefulSet and a subsequent regular redeploy restores the image from `opsroot.json`. Both scripts push images in `--buildx` mode. [publish.sh](publish.sh) pushes Git refs. See [spec/build.md](spec/build.md) before using these scripts.
