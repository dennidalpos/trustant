# Upstream attribution

This directory is vendored in the Trustant repository (it was the `acp`
submodule). It is not a clean-room work: it starts from an upstream project and
adds Trustant code.

## obsidian-agent-client (the origin of `acp/`)

- Upstream: https://github.com/RAIT-09/obsidian-agent-client
- Baseline: tag `0.11.0`. The first import of this code (75 of its 119 files)
  matched that tag byte for byte.
- License: Apache License 2.0, Copyright 2025-2026 RAIT-09. The upstream license
  text is kept in [LICENSE-Apache-2.0-obsidian-agent-client](LICENSE-Apache-2.0-obsidian-agent-client)
  and the upstream NOTICE is reproduced in [NOTICE](NOTICE).
- Trustant's own additions and changes are licensed under AGPL-3.0-or-later
  ([LICENSE](LICENSE)). Apache-2.0 allows the combined work to be distributed
  under the AGPL; the Apache-2.0 terms still apply to the upstream parts.

How the Apache-2.0 obligations are met:

| Obligation | Where |
|---|---|
| Give recipients a copy of the license | `LICENSE-Apache-2.0-obsidian-agent-client` and the root `LICENSE` |
| Keep copyright and attribution notices | `NOTICE`, and a "Derived from obsidian-agent-client" line in the header of each derived source file |
| Carry the upstream NOTICE | `NOTICE` |
| Mark changed files prominently | The header line says "modified by Nuvolaris Inc."; the per-file list is [../legal/VENDORED.md](../legal/VENDORED.md) |

## pi-acp (`pi-acp/`)

A fork of https://github.com/svkozak/pi-acp at commit
`9e857dcc05a057404eb1537e5f31e5aef88a5863` (`pi-acp@0.0.31`), MIT, Copyright (c)
2025 Sergii Kozak. Its `LICENSE` is kept unchanged and no AGPL header is added to
its files. Changes are described in [pi-acp/TRUSTANT_FORK.md](pi-acp/TRUSTANT_FORK.md)
and listed per file in [../legal/VENDORED.md](../legal/VENDORED.md).

## Restoring the original submodules

`acp/` and `acp/pi-acp/` were submodules before they were vendored (commit
`b06dc8d`). Their last submodule commits stay reachable in this repository's
history as parents of that commit, so they must not be removed by a history
rewrite:

| Directory | Original repository | Last submodule commit |
|---|---|---|
| `acp/` | https://github.com/trustant/trustant-acp.git | `9786f796380aa9b2000d8cebd913fa41150c89ed` |
| `acp/pi-acp/` | https://github.com/trustant/pi-acp | `4e526e5f64eaec647d78c919ba9dc071197c2517` |

If the submodules are ever restored, [../legal/VENDORED.md](../legal/VENDORED.md)
lists every file changed since then ("Delta against the original submodules").
Export the changes as a patch to apply on the original repositories:

```bash
git diff --binary 9786f796380a^{tree} HEAD:acp > acp-since-submodule.patch
git diff --binary 4e526e5f64ea^{tree} HEAD:acp/pi-acp > pi-acp-since-submodule.patch
```

(The first patch also contains the `pi-acp/` files; restrict it with a path
filter if needed.) Do not drop the Apache-2.0 attribution when restoring: it
belongs to the upstream project, not only to this repository.

## Tracking changes

`python legal/vendored_changes.py` compares every file here with the baseline and
rewrites `legal/VENDORED.md`. Run it whenever files in this directory change.
