# Changelog

All notable changes to Trustant are listed here. The latest release tag is
`trustant_v1.0.3_26.324.1910`; entries below it are not released yet.

## Unreleased

### Added

- The Ollama Instance card detects an Ollama on the host of a WSL or Lima
  environment (default gateway, or `host.lima.internal`), checks that it answers
  and offers it with an **Add** button. It only suggests: the switch stays off
  until you turn it on.
- `OLLAMA_EXTERNAL=1` makes `start.sh` skip installing and enabling the local
  Ollama service, so an Ollama on another machine (for example the Windows
  app under WSL) keeps `localhost:11434`. The default is unchanged.

### Changed

- The OpenCode end-to-end scripts (`tests/e2e_*.sh`, `issue98.spec.mjs`,
  `issue98-session-tree.mjs`), the provider benchmarks built on them and their
  diagrams were removed with the OpenCode runtime; `npm run e2e:*` and
  `test:e2e-providers` no longer exist. See [spec/8-e2e.md](spec/8-e2e.md).
- `oplugins/` (trustable-ai/openserverless-task), `skills/` (trustable-ai/skills)
  and `mcp/` (trustant/openserverless-mcp) are now vendored directories instead
  of Git submodules. Their `LICENSE` and `NOTICE` files are unchanged and their
  differences from the upstream baselines are tracked in
  [legal/VENDORED.md](legal/VENDORED.md). `oplugins-truinst` remains the only
  submodule because the release scripts push to it.
- `oplugins/prereq.yml` carries the pinned and verified Linux provisioning tool
  downloads (helm 3.22.1, kubectl 1.33.13, kind 0.33.0, yq 4.54.3).
- `start.sh` and `setup.sh` no longer initialise `mcp` as a submodule and only
  verify that the vendored sources exist; `image/image.sh` stages `mcp/` from
  the working tree and prints a content hash of what it staged.
- `DEPS`, `LICENSE` and `NOTICE` were regenerated: `@tanstack/react-virtual` is
  gone, and React 19.3.1, ACP SDK 2.9.1 and diff 9.0.0 are listed.

### Fixed

- A failed model check offers a direct link to change the model or host, and
  displays the provider's error message in plain text.
- Application deletion retries OpenServerless cleanup automatically and can
  finish when some cluster resources were already removed.
- Model configuration rows keep their actions together and use consistently
  sized status badges and buttons. **Hide again** is available after overriding
  a model's size recommendation.
- Tooltip question marks and checkboxes align with the surrounding text and
  match its size.
- Configuration displays the provider and model settings together after loading.
  Long Ollama instance names and addresses fit the panel on narrow screens.
- The app inventory distinguishes status from links, increases link click areas,
  and suppresses the default Welcome banner.
- Inventory and editor dialogs expose labels for their fields. Escape dismisses
  the license and Route dialogs and restores focus to their openers.
- The editor switches between the chat and preview at narrow widths and keeps
  both frames loaded. The frames have names that describe their contents.
- Save & Configure no longer stops with "Ollama hostname is required" when an
  external Ollama instance is active: the hand-typed host check now applies only
  when no saved instance owns the endpoint.
- The Private AI connection test allows up to 180 seconds for cold starts and
  distinguishes timeouts from unreachable endpoints and incomplete responses.
- Configure warns beside the Pi model menu when the saved Private AI model list
  could not be refreshed.

### Security

- `mcp/`: `zod` 4.6.7 (GHSA-mrhx-c7xc-m3w5) and the Python pins certifi
  2026.10.18, dnspython 2.9.2, langgraph 1.2.14, langgraph-sdk 0.4.6, nltk
  3.10.4, pymongo 4.18.3 and soupsieve 4.0 resolve the open Dependabot alerts
  for those packages. `mcp` typecheck and tests pass on Linux (16/16).
- `oplugins/ide/deploy`: `glob` 14.0.2 with a regenerated `bun.lock` (Bun 1.4.5);
  its test and a `glob()` call pass.

- Git over SSH no longer disables host-key checking: `gitSSHCommand` uses
  `StrictHostKeyChecking=accept-new` with `~/.ssh/known_hosts_trustant`, so a
  host is trusted on first use and a changed key is refused. To accept a
  legitimately rotated key, delete its line from that file.

- The managed `ops ide devel` watcher starts without a shell: `opsDevelCommand`
  sets the working directory instead of interpolating the path into `sh -c`.

- `ops admin adduser` receives the generated password in `OPS_ADMIN_PASSWORD`
  instead of on the command line (`addOpsUser`), so it no longer shows in the
  process list. The vendored `oplugins/admin` accepts it; an older installed
  `ops` is detected by its usage error and retried with the positional form
  until the updated plugin is published (#100).

- `TRUSTANT_AUTH_TRUSTED_PROXIES` (IPs or CIDRs) lets the login rate limiter use
  the client address from `X-Forwarded-For` when the request comes from a listed
  proxy. Unset keeps the transport-peer behaviour.

- The server and `run.sh` warn at startup while authentication is disabled,
  because port 8910 listens on all interfaces over plain HTTP.

- Line injection into the env-style files is closed. `writeEnvFile`,
  `saveSharedEnv` and `writeEnvDistBindings` refuse names, values and patterns
  containing a newline, carriage return or NUL, and `POST /api/appconfig`,
  `POST /api/shared` and `POST /api/imports` answer 400. Variable names in
  `POST /api/appconfig` must now match `[A-Za-z_][A-Za-z0-9_]*`.

### Tests

- Added tests for session key parsing, the session and logout endpoints, host
  name parsing and host-based routing and redirects. Statement coverage of the
  Go package went from 56.5% to 57.2%.

### Known issues

- Local authentication is disabled unless `TRUSTANT_AUTH_MODE=local` is set, and
  the `truacp.*`, `vite.*` and application hosts bypass it; see
  [spec/authentication.md](spec/authentication.md).
