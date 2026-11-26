# Changes made by the Trustant project

This directory is a vendored copy of https://github.com/trustable-ai/openserverless-task
(Apache-2.0, baseline commit `27b281eb7615c9094f1c6edab10a2b4fe2fd621c`). The upstream
LICENSE and NOTICE are unchanged. As required by Apache-2.0 section 4(b), these files
were modified by the Trustant project:

- `admin/docopts.md`, `admin/opsfile.yml`: admin password can be passed through `OPS_ADMIN_PASSWORD`.
- `debug/runtime/invoke.py`: payload handling and exception clean-up.
- `ide/deploy/package.json`, `ide/deploy/bun.lock`: dependency update (glob).
- `prereq.yml`: pinned, checksum-verified downloads of the Linux Kubernetes tools.
- `setup/kubernetes/DEBUG.txt`: documentation change.
- `util/upload/.env.test`: removed.

The per-file registry is [../legal/VENDORED.md](../legal/VENDORED.md).
