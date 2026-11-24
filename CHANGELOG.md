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

### Tests

- Added tests for session key parsing, the session and logout endpoints, host
  name parsing and host-based routing and redirects. Statement coverage of the
  Go package went from 56.5% to 57.2%.

### Known issues

- `mcp/` depends on a `zod` version affected by GHSA-mrhx-c7xc-m3w5 (memory
  exhaustion on very large constrained collections); the upgrade to a fixed
  major version is tracked in the project tracker.
- Local authentication is disabled unless `TRUSTANT_AUTH_MODE=local` is set, and
  the `truacp.*`, `vite.*` and application hosts bypass it; see
  [spec/authentication.md](spec/authentication.md).
