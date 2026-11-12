# 20 - Legal texts: DEPS, LICENSE, NOTICE

Trustant is AGPL-3.0-or-later. Every Trustant build carries three plain-text
files at the repo root:

| File | Content |
|---|---|
| `DEPS` | Inventory of declared component inputs: name, version, license (SPDX expression), homepage, grouped by how it reaches the user |
| `LICENSE` | Trustant's AGPL-3.0 text first, then the full text of **every** license identifier that appears in `DEPS`, each under a `License: <id>` header |
| `NOTICE` | Trustant's own notice (`legal/NOTICE.head`), then per component the original license files and NOTICE files, with extracted copyright lines for navigation |

The generator also writes `legal/COMPONENTS.json`: component/version/license,
source selector, evidence status, legal-file text hashes and obligations to
review. Its statuses distinguish collected files from declared-only metadata;
they are not legal clearance. Submodule entries include the checkout commit.

## Generation: `./deps.sh`

The three files are **generated, never edited by hand**. `deps.sh` runs
`legal/deps.py`, which reads:

- `legal/deps.manual.txt` — hand-maintained components (vendored web assets,
  submodules, Ubuntu packages, downloaded binaries, ops prerequisites, global
  npm and Python tools, the license of each pinned Python package, dev-only
  tools). Format and fetch kinds are documented at the top of the file.
- `go.mod` — Go modules (license files from the Go module cache).
- The production trees (`dev != true`) of `acp/`, `acp/pi-acp/`, `mcp/` and
  `react-mcp/` `package-lock.json` — license from the lockfile, files from the
  local `node_modules` or `npm pack`.
- `acp/pi.version` — the coding agents installed globally in the image.
- `acp/extensions/requirements.txt` — versions of the Python packages; the
  generator fails when a pinned package has no line in `deps.manual.txt`.
- `oplugins/prereq.yml` — versions of the binaries `ops` downloads.

Components' own LICENSE/NOTICE files are downloaded once into `legal/.cache`
(gitignored; `./deps.sh --clean` empties it). License texts live in
`legal/licenses/<id>.txt` and are committed: a missing SPDX text is fetched from
`spdx/license-list-data`; `LicenseRef-*` texts are written by hand.

Downloads use at most six workers, coalescing identical sources so two npm
trees cannot write the same cache concurrently. `npm pack` runs with
`--ignore-scripts`; package lifecycle hooks are not needed to read legal
metadata. A missing legal source aborts generation before any generated
outputs is replaced. All license texts are resolved before writing, and output
uses UTF-8 with LF line endings. Download failures are errors; packages without
a license file still produce a warning and need explicit review.
Trailing spaces in the original texts are preserved; `.gitattributes` disables
only end-of-line whitespace warnings for the generated NOTICE file.

`OFL.txt` is recognized as a license file. Fonts referenced by
`web/trustant-ui.css` are listed separately as browser downloads, not as files
bundled in the image. Their Google Fonts responses are not version-pinned.

Sections of DEPS: Go binary, vendored web assets, submodules, the four npm
trees, Ubuntu packages, downloaded binaries, ops prerequisites, global npm
installs (top level only — their trees ship with their license files in the
image's global `node_modules`), Python tools, Python packages for user apps,
and development-only tools (listed, not distributed, no notices collected).

This inventory describes build inputs, not the complete contents of a resolved
container image. Floating versions, indirect Ubuntu packages, global npm/Python
dependency trees, downloaded runtime libraries and artwork provenance need
separate verification. See [review scope and open items](../legal/REVIEW.md).

Claude Code is deliberately absent: it is proprietary and is never shipped
(issue #12); a note in DEPS says so.

## Embedding and API

[legal.go](../legal.go) embeds the three files with `//go:embed`.

`GET /api/legal/{deps,notice,license}` returns the text as
`text/plain; charset=utf-8`. Unknown names are 404; methods other than GET or
HEAD return 405.

## UI

`applist.html` has a footer with a **License** link. It opens a modal with three
tabs — DEPS, NOTICE, LICENSE — each lazily fetching its endpoint once and
showing it in a scrollable monospace block. Esc, the ✕ button or a click on the
backdrop closes it.

## Container image

`build.sh` (`--build` and `--buildx`) and `hotfix.sh` copy the three files into
`image/` (gitignored copies); `image/Dockerfile` and the hotfix layer copy them
to `/usr/share/doc/trustant/`.

## Tests

[legal_test.go](../legal_test.go): the endpoints serve the embedded texts;
every license id in a DEPS table row has a `License: <id>` section in LICENSE;
NOTICE and LICENSE start with Trustant's own texts; build.sh, hotfix.sh and the
Dockerfile ship the files into the image.

`python -B -m unittest discover -s tests -p test_legal_deps.py` checks that
unavailable sources and missing license texts preserve existing outputs, that
shared sources are fetched once, and that original multiline license notices
survive generation. These checks use fixtures and do not fetch dependencies.
