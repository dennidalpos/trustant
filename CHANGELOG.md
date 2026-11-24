# Changelog

All notable changes to Trustant are listed here. The latest release tag is
`trustant_v1.0.3_26.324.1910`; entries below it are not released yet.

## Unreleased

### Changed

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

### Tests

- Added tests for session key parsing, the session and logout endpoints, host
  name parsing and host-based routing and redirects. Statement coverage of the
  Go package went from 56.5% to 57.2%.

### Known issues

- Local authentication is disabled unless `TRUSTANT_AUTH_MODE=local` is set, and
  the `truacp.*`, `vite.*` and application hosts bypass it; see
  [spec/authentication.md](spec/authentication.md).
