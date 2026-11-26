# AGENTS.md


## Scope and authorization

- Work within the user's request. Audits and reviews are read-only. For substantial changes, make a short plan and use an existing issue when the user provides one.
- Create or edit GitHub issues, PRs, and wiki pages only when explicitly requested. Commits, pushes, merges, releases, deployments, and destructive operations each require explicit authorization for the relevant scope. Check what scripts do before running them.
- Preserve unrelated work and keep credentials in their authentication stores. Before publishing, verify the destination and scan the working tree, index, and relevant history for secrets and private data. Follow [CONTRIBUTING.md](CONTRIBUTING.md) for an authorized PR.
- Third-party licenses: before adding, vendoring, copying, or modifying third-party code, assets, or text, read the license and any NOTICE of the exact upstream version, and check compatibility with AGPL-3.0-or-later. Copy license texts, copyright notices, NOTICE content, and attribution exactly as that license requires (for example Apache-2.0 section 4: keep a copy of the license, retain notices, mark changed files prominently, carry the upstream NOTICE; MIT: keep the copyright and permission notice). Never strip, rewrite, or relicense upstream notices, and never put this project's license header on code that stays under another license. Record every component in [legal/deps.manual.txt](legal/deps.manual.txt) and [CREDITS.md](CREDITS.md). If compatibility or the required notices are unclear, stop and ask the owner.
- Vendored sources: `acp/` (derived from obsidian-agent-client 0.11.0, Apache-2.0, Copyright RAIT-09), `acp/pi-acp/` (fork of svkozak/pi-acp 0.0.31, MIT), `oplugins/` (trustable-ai/openserverless-task, Apache-2.0), `skills/` (trustable-ai/skills, Apache-2.0) and `mcp/` (trustant/openserverless-mcp, Apache-2.0) were integrated from former submodules/gitlinks; their LICENSE and NOTICE files stay unchanged. Their changes are monitored and tracked in [legal/VENDORED.md](legal/VENDORED.md); regenerate it with `python legal/vendored_changes.py` whenever files under `acp/`, `oplugins/`, `skills/` or `mcp/` change, and regenerate [UPSTREAM-COMPARISON.md](UPSTREAM-COMPARISON.md) with `python legal/upstream_compare.py` (comparison with the original repository and its gitlinks; add a `REASONS` entry in the script for each new modified file; `--check` verifies it is current), and review any file whose status moves from unmodified to modified. The same file also records the delta against the original submodule commits, so the submodules can be restored later if requested (see [acp/UPSTREAM.md](acp/UPSTREAM.md)); keep those commits reachable in history.
- Keep [CREDITS.md](CREDITS.md) in step with component changes and regenerate DEPS, LICENSE, and NOTICE with `./deps.sh` (see [spec/20-legal.md](spec/20-legal.md)); they are generated, never edited by hand.
- The only remaining gitlink is `oplugins-truinst` ([.gitmodules](.gitmodules), release repository). Follow its own instructions, update it only when the change requires, and push to it only with explicit authorization. Never push to third-party repositories: a change to third-party code is made in the vendored copy here and tracked in [legal/VENDORED.md](legal/VENDORED.md).

## Changes and verification

- Verify current behavior in code and tests. Use [spec/](spec/) for intended behavior and update the relevant spec when behavior changes. Flag conflicts between the spec and the implementation. For a new API, register its route in [main.go](main.go) and keep its handler with the feature.
- Use `./start.ps1` from PowerShell on Windows (WSL2) or `./start.sh` on macOS (Lima) and native Ubuntu. Run `./run.sh` in the Ubuntu environment. See [spec/start.md](spec/start.md) and [spec/run.md](spec/run.md). Use [.env.dist](.env.dist) and [spec/0-preflight.md](spec/0-preflight.md) for environment setup.
- Run relevant checks. The Go suite (`go test ./...`) requires the ignored `_build.txt` embedded by [main.go](main.go). The development loop generates it before starting Air. Node checks and E2E entry points are listed in [package.json](package.json). Cluster scenarios may launch apps or call model providers. See [spec/8-e2e.md](spec/8-e2e.md).
- Review the diff and `git status --ignored` before handing off. Report checks run, failures, and unverified behavior.

## Trustant conventions

- Use **Trustant** for the product. This repository and its submodules are maintained as the owner's forks; the owner does not track or merge upstream changes, so do not assume upstream updates, rebase onto upstream, or propose syncing from it unless asked. Preserve the module path `github.com/trustant/trustant` in [go.mod](go.mod), which imports depend on. Change the URLs in [.gitmodules](.gitmodules) to a fork only when the owner asks. TruACP (`acp/`, including `acp/pi-acp/`) is vendored in this repository and changes with it; it is not a submodule; the same holds for `oplugins/`, `skills/` and `mcp/`.
- The durable app repository is bare and lives at `$WORKSPACE_DIR/workspace/<name>/`. The active checkout lives at `$WORKBENCH_DIR/<name>/` and receives regenerated configuration on launch. See [spec/4-launch.md](spec/4-launch.md) and [spec/13-gitignore.md](spec/13-gitignore.md).
- Use an FQDN for checks that exercise [hostname routing](middleware.go). The current host labels are `trustant` for the UI/API, `truacp` for the assistant, and `vite` for the app preview. Registered app hosts route to their development services.
- Treat [openserverless-instructions.md](openserverless-instructions.md) and [app-agents.md](app-agents.md) as product assets that guide generated apps. Preserve the license error prefixes consumed by the UI when changing that API. See [spec/14-license.md](spec/14-license.md).

## Release script side effects

- [build.sh](build.sh) in `--tag` and `--build` modes replaces local Git tags, updates `oplugins-truinst/opsroot.json`, and attempts `git commit -a`. With `--build --no-deploy`, those effects still occur. The regular deploy reads the image from `opsroot.json`.
- [hotfix.sh](hotfix.sh) can also replace local tags. Its deploy patches the live StatefulSet and a subsequent regular redeploy restores the image from `opsroot.json`. Both scripts push images in `--buildx` mode. [publish.sh](publish.sh) pushes Git refs. See [spec/build.md](spec/build.md) before using these scripts.

## Git workflow (push policy)

- The repository owner is an admin and changes are verified locally before any push. When a push is authorized, push directly to `main` and do not open a pull request. This applies only to pushes the user has authorized; it does not authorize committing or pushing on its own. If anything is unclear (failed checks, unexpected remote state, protected-branch rejection), stop and ask.

<!-- secondbrain:begin -->
## SecondBrain

This repository's private context, decisions and tasks live in SecondBrain, outside the repository.

- At the start of a session run `sb context` (MCP tool `get_context`) and work from it.
- Run `sb start` to check recorded checkouts, refresh managed agent files and see which SecondBrain build is running.
- When asked to run `sb start` in chat, include the full result in the first reply after execution, even in a new session. Use bullet lists for the executable and PATH status, repository, context and tasks, instruction alignment, MCP configuration, health findings, all suggested next steps and every command shown. Under each command, list its parameters and what they do. State when no issues are reported and distinguish MCP configuration from runtime verification.
- When asked to onboard a repository with missing or empty context, read `sb guide onboarding` or call `get_guide` with topic `onboarding`. Inspect the checkout and save the verified starting context in the private dossier.
- When asked to work through tasks in the current chat, read `sb guide chat` or call `get_guide` with topic `chat`.
- Track work with `sb task add`, `sb task list` and `sb task done` (MCP task tools).
- Record durable decisions with `sb decide "..."` and keep the current state with `sb context --set`.
- Never copy dossier content into the repository.
<!-- secondbrain:end -->
