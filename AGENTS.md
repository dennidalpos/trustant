# AGENTS.md

<!-- secondbrain:repository:begin -->
Common directives applied: revision **1.4** — 2026-11-14. <!-- secondbrain:source global=C9BEBD22A7BC4A859D2C498D88D98DFDEC32BF5A5DAB373CEF3E9C40E235B06F workstation=BD2E210B49B3B77000F570E6F6191DC694798722B50900E6C3864E3DBDD78FF7 -->
## SecondBrain

Repository knowledge lives in the SecondBrain dossier; code, guides, and publishable assets stay here.

- Load the slim startup with the SecondBrain checkout's scripts/knowledge.ps1 -Startup or MCP get_startup_context, and follow its references for detail. If the root is unreachable, report the dossier as unavailable.
- Write only to this repository's dossier `repos/<repo>/`, only through SecondBrain scripts or MCP write tools. Work folders come from scripts/work.ps1 -Init; sessions, retention, and cleanup rules load from `sb prompt`.
- Commits, pushes, deletions, and publication require specific authorization.
<!-- secondbrain:repository:end -->

## Scope and authorization

- Work within the user's request. Audits and reviews are read-only. For substantial changes, make a short plan and use an existing issue when the user provides one.
- Create or edit GitHub issues, PRs, and wiki pages only when explicitly requested. Commits, pushes, merges, releases, deployments, and destructive operations each require explicit authorization for the relevant scope. Check what scripts do before running them.
- Preserve unrelated work and keep credentials in their authentication stores. Before publishing, verify the destination and scan the working tree, index, and relevant history for secrets and private data. Follow [CONTRIBUTING.md](CONTRIBUTING.md) for an authorized PR.
- Keep [CREDITS.md](CREDITS.md) in step with component changes and regenerate DEPS, LICENSE, and NOTICE with `./deps.sh` (see [spec/20-legal.md](spec/20-legal.md)); they are generated, never edited by hand.
- Use [.gitmodules](.gitmodules) and follow the instructions inside each affected submodule. Update only the submodules and gitlinks required by the change. Push to plugin repositories only with explicit authorization for those repositories.

## Changes and verification

- Verify current behavior in code and tests. Use [spec/](spec/) for intended behavior and update the relevant spec when behavior changes. Flag conflicts between the spec and the implementation. For a new API, register its route in [main.go](main.go) and keep its handler with the feature.
- Use `./start.ps1` from PowerShell on Windows (WSL2) or `./start.sh` on macOS (Lima) and native Ubuntu. Run `./run.sh` in the Ubuntu environment. See [spec/start.md](spec/start.md) and [spec/run.md](spec/run.md). Use [.env.dist](.env.dist) and [spec/0-preflight.md](spec/0-preflight.md) for environment setup.
- Run relevant checks. The Go suite (`go test ./...`) requires the ignored `_build.txt` embedded by [main.go](main.go). The development loop generates it before starting Air. Node checks and E2E entry points are listed in [package.json](package.json). Cluster scenarios may launch apps or call model providers. See [spec/8-e2e.md](spec/8-e2e.md).
- Review the diff and `git status --ignored` before handing off. Report checks run, failures, and unverified behavior.

## Trustant conventions

- Use **Trustant** for the product. This repository and its submodules are maintained as the owner's forks; the owner does not track or merge upstream changes, so do not assume upstream updates, rebase onto upstream, or propose syncing from it unless asked. Preserve the module path `github.com/trustant/trustant` in [go.mod](go.mod), which imports depend on. Change the URLs in [.gitmodules](.gitmodules) to a fork only when the owner asks. TruACP (`acp/`, including `acp/pi-acp/`) is vendored in this repository and changes with it; it is not a submodule.
- The durable app repository is bare and lives at `$WORKSPACE_DIR/workspace/<name>/`. The active checkout lives at `$WORKBENCH_DIR/<name>/` and receives regenerated configuration on launch. See [spec/4-launch.md](spec/4-launch.md) and [spec/13-gitignore.md](spec/13-gitignore.md).
- Use an FQDN for checks that exercise [hostname routing](middleware.go). The current host labels are `trustant` for the UI/API, `truacp` for the assistant, and `vite` for the app preview. Registered app hosts route to their development services.
- Treat [openserverless-instructions.md](openserverless-instructions.md) and [app-agents.md](app-agents.md) as product assets that guide generated apps. Preserve the license error prefixes consumed by the UI when changing that API. See [spec/14-license.md](spec/14-license.md).

## Release script side effects

- [build.sh](build.sh) in `--tag` and `--build` modes replaces local Git tags, updates `oplugins-truinst/opsroot.json`, and attempts `git commit -a`. With `--build --no-deploy`, those effects still occur. The regular deploy reads the image from `opsroot.json`.
- [hotfix.sh](hotfix.sh) can also replace local tags. Its deploy patches the live StatefulSet and a subsequent regular redeploy restores the image from `opsroot.json`. Both scripts push images in `--buildx` mode. [publish.sh](publish.sh) pushes Git refs. See [spec/build.md](spec/build.md) before using these scripts.

## Git workflow (push policy)

- The repository owner is an admin and changes are verified locally before any push. When a push is authorized, push directly to `main` and do not open a pull request. This applies only to pushes the user has authorized; it does not authorize committing or pushing on its own. If anything is unclear (failed checks, unexpected remote state, protected-branch rejection), stop and ask.
