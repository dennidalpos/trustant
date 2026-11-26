# Image qualification: trustant_v1.0.3_26.324.1910

Date: 2026-11-26. Status: **partial**. This is a static, read-only analysis by
digest: no container was started, nothing was deployed or published. Layers
were downloaded to the ignored `tmp/` and each layer's SHA-256 was checked
against its manifest. It is a technical inventory, not a legal opinion.

## Digests

| Item | Digest |
|---|---|
| Image index `ghcr.io/trustant/trustant:trustant_v1.0.3_26.324.1910` | `sha256:713e26c2a5563a43d5366ee2d84752ada3788e07fe27f8238c09b619804a62ab` |
| linux/amd64 manifest (35 layers, about 1.2 GB) | `sha256:95174b2f742fa4cfc895f51e2ffa2a4c31c79971866eab8dfc9baf1ac4303295` |
| linux/arm64 manifest | `sha256:541649c6d43a72bcc3e80164132f381d579f3e6f850ae30be2d0844d33fd52e8` |
| amd64 / arm64 attestations | `sha256:4ce94480…1faf` / `sha256:b8e6514d…fb0a` (SLSA provenance only) |

The registry publishes SLSA provenance but **no SBOM**, so the inventory below
comes from the layer contents. It replaces the v1.0.2 digests recorded earlier.

## Inventory (identical on both architectures)

| Ecosystem | Entries | Notes |
|---|---|---|
| Ubuntu 24.04 packages (dpkg) | 196 | 187 have `/usr/share/doc/<pkg>/copyright`; 9 do not (libgcc-s1, libncursesw6, libstdc++6, libatomic1, libpython3.12-stdlib, libpython3.12t64, openssh-server, openssh-sftp-server) |
| npm packages (global trees) | 1216 amd64, 1215 arm64 | only platform-specific native packages differ (esbuild, rollup, bun, codex, recheck, typescript, atlas-local) |
| Python distributions | 145 | same on both |
| License/NOTICE files found | 1235 | in `node_modules`, `site-packages`, `usr/local`, `opt` |
| Large executables | trustant, ollama, gh, mcp-s3 (+ copy mcp-s3-real) | plus native libraries in the uv cache |

## Findings

1. **Register drift.** DEPS lists build inputs, not the resolved image
   (already stated in [REVIEW.md](REVIEW.md)). Against the image: 165 of 196
   apt packages, 738 of 1216 npm entries and 93 of 145 Python distributions
   are not listed by name in DEPS. Of the 50 Python distributions that are
   listed, 33 have a different version in the image (for example psycopg
   3.3.9 vs 3.3.5, orjson 3.13.0 vs 3.12.0, certifi 2026.10.23 vs 2026.10.18).
   The registers need regenerating from a resolved lock, or a note that DEPS is
   a top-level list only.
2. **Copyleft inside the image.** Python: `pglast` 7.21 (GPL-3.0-or-later,
   bundles libpg_query), `psycopg` / `psycopg-binary` / `psycopg-pool` 3.3.x
   (LGPL-3.0-only), `docstring-to-markdown` 0.17 (LGPL-2.1-or-later);
   MPL-2.0: certifi, orjson, pathspec. pglast, psycopg and certifi/orjson are
   already declared in DEPS; `docstring-to-markdown` and `psycopg-pool` are not.
   apt: 128 packages carry GPL/LGPL/MPL stanzas (coreutils, bash, git, gnupg,
   curl and so on); corresponding source is available from the Ubuntu archive
   and should be offered on request.
3. **Missing or non-SPDX license metadata.** npm: `cli-table` 0.3.11,
   `openserverless-mcp` 0.1.0 and `trustant-react-mcp` 0.1.0 declare none
   (the last two are project components); `argparse` 2.0.1 is Python-2.0,
   `npm` 11.19.1 Artistic-2.0, `spdx-exceptions` CC-BY-3.0 (attribution).
   Python: `mcp-server-milvus` and `redis-entraid` declare none; `pathspec`
   only by classifier (MPL-2.0).
4. **Binaries.** The ollama (0.40.18 in DEPS), gh (2.105.0) and mcp-s3 (1.3.0)
   versions in DEPS were not confirmed against the image binaries because
   nothing was executed. The `trustant` binary has no DEPS row; its source is
   this repository.

## Residual (why this is partial)

- Version of the shipped binaries (ollama, gh, mcp-s3) not proven.
- Corresponding sources for the GPL/LGPL components not retrieved or linked.
- Licenses of the 3 unlicensed npm and 2 unlicensed Python entries not yet
  resolved; 9 apt packages without a copyright file.
- DEPS / COMPONENTS.json stay a top-level list (decision 2026-11-26); this
  file is the full resolved inventory.

## Follow-up 2026-11-26 (amd64 layers re-read, nothing executed)

- `psycopg-pool` 3.3.3 (LGPL-3.0-only, in the postgres-mcp tool env) and
  `docstring-to-markdown` 0.17 (LGPL-2.1-or-later, in python-lsp-server) are
  now in DEPS and NOTICE.
- **`gh` in the image is 2.96.0** (build string in `usr/local/bin/gh`, layer 6),
  while `image/Dockerfile` and DEPS pin 2.105.0. DEPS describes the current
  repository, not v1.0.3; the next image release should carry 2.105.0.
- `mcp-s3`: `go version -m` (reads the embedded build info, nothing executed)
  gives module `github.com/txn2/mcp-s3` v1.3.0, commit `872fa60a`, built with
  go1.26.4. This matches DEPS (1.3.0).
- `ollama`: Go 1.26.0 build whose embedded info says `(devel)`, with no version
  string recoverable from the binary. The shipped version stays unproven (DEPS
  pins 0.40.18); confirm it from the image build log or by running the binary.
- Corresponding sources, from the pinned PyPI sdists: pglast 7.21
  (`pglast-7.21.tar.gz`, github.com/lelit/pglast), psycopg 3.3.x and
  psycopg-pool 3.3.3 (github.com/psycopg/psycopg), docstring-to-markdown 0.17
  (github.com/python-lsp/docstring-to-markdown). For the apt packages the
  source is `apt-get source <pkg>=<version>` from the Ubuntu 24.04 archive; the
  source offer for the image is still to be written.
- Licenses of the entries without metadata, read from the upstream files:
  `cli-table` 0.3.11 MIT (LICENSE in the npm tarball, (c) 2010 LearnBoost),
  `redis-entraid` 1.2.2 MIT (LICENSE in the wheel, (c) Redis Inc.),
  `mcp-server-milvus` Apache-2.0 (classifier and upstream LICENSE; the sdist
  does not ship the file). These are not yet in DEPS/NOTICE.
- Still open: the 9 apt packages without a copyright file, the written source
  offer, and adding the three licenses above to the registers.

Reproduce: resolve the tag through the registry API, take the per-arch manifest
digests above, download and verify layers, scan tar members for
`var/lib/dpkg/status`, `usr/share/doc/*/copyright`, `package.json` under
`node_modules` and `*.dist-info/METADATA`.

## Update 2026-11-27

- The 9 apt packages without their own copyright file have their documentation
  directory symlinked to a package that has one: libatomic1, libgcc-s1 and
  libstdc++6 to gcc-14-base; libncursesw6 to libtinfo6; libpython3.12-stdlib to
  libpython3.12-minimal (libpython3.12t64 to libpython3.12-stdlib);
  openssh-server and openssh-sftp-server to openssh-client. Nothing is missing.
- Python LGPL components installed: psycopg 3.3.9, psycopg-binary 3.3.9,
  psycopg-pool 3.3.3 (all LGPL-3.0-only); now in DEPS.
