# Third-party licensing review

Review date: 2026-11-12. Scope: source checkout and the declared Linux container
build inputs. This is a technical inventory, not a legal opinion or clearance
of a published image.

## Registers and evidence

- [DEPS](../DEPS) lists 480 build, runtime and development inputs by component,
  version, declared SPDX expression and upstream homepage.
- [COMPONENTS.json](COMPONENTS.json) records each input's source selector,
  evidence status, normalized-text SHA-256 hashes of collected legal files,
  and obligations to review. Submodule records include the checked-out commit.
- [NOTICE](../NOTICE) preserves collected component license and NOTICE texts,
  including component-specific conditions and multiline attributions.
- [LICENSE](../LICENSE) starts with Trustant's AGPL text and includes the 32
  SPDX/license-reference texts named in DEPS. An SPDX text does not replace a
  component's own copyright notice.

The generator collected license files for 445 entries. Another 26 entries have
only declared metadata: the 25 direct Ubuntu/build-stage entries and the
Windows-only zip helper. Nine entries are development-only. These are inventory
entries, not unique installed packages; dependencies may occur in several trees.
`license-files-collected` means the source file was retrieved, not that every
upstream licensing statement was legally validated.

The 305 npm production-tree entries come from the four committed lockfiles.
The 87 pinned Python application packages were checked against PyPI release
metadata: 45 supplied SPDX expressions, all matching the declared expressions;
the other 42 need their legacy metadata and original files to be interpreted.
This comparison does not cover every dependency of global tools or binaries.

## Corrections to the registers

| Component | Version/source | SPDX expression | Status and obligation |
|---|---|---|---|
| Trustant | Repository source; headers and NOTICE.head | AGPL-3.0-or-later | Declaration agrees with LICENSE and .licenserc.yaml; review source delivery for binaries and modified network deployments. |
| skills | Commit recorded in COMPONENTS.json | AGPL-3.0-or-later | Corrected the former Apache-2.0 entry to match the checked-out LICENSE. Preserve the license; review any copied skills separately. |
| 7-Zip / 7zz | 2407; upstream 24.07 source license | LGPL-2.1-or-later AND BSD-3-Clause AND BSD-2-Clause AND LicenseRef-unRAR | Added the BSD-2-Clause portion and used the release-specific text. Review the unRAR restriction and source obligations; do not infer compatibility from the LGPL identifier alone. |
| Work Sans | Unpinned Google Fonts response | OFL-1.1 | Added the browser-fetched font and upstream copyright/license. Font binaries are not stored in the image. |
| JetBrains Mono | Unpinned Google Fonts response | OFL-1.1 | Added the browser-fetched font and upstream copyright/license. Font binaries are not stored in the image. |
| packages / openserverless-packages | Absent from .gitmodules and image staging | Previously Apache-2.0 | Removed the stale submodule entry. This says nothing about application-level packages downloaded independently. |
| zip helper | Windows-only ops prerequisite | Info-ZIP | Separated from the prerequisites used by the Linux image. |

The complete package/version/license/status/obligation list is in
[COMPONENTS.json](COMPONENTS.json). The hand-maintained selectors are in
[deps.manual.txt](deps.manual.txt); regenerate rather than editing output texts.

## Compatibility and attribution review

Permissive declarations in the Go and npm trees do not by themselves identify a
conflict with the project's AGPL declaration. Preserve the applicable MIT, ISC,
BSD and Apache notices and license terms. Apache-2.0 also specifies preservation
of relevant NOTICE attributions and notices of modifications. This review does
not establish the legal boundary of any combined work.

| Components | Review required before distributing a resolved image |
|---|---|
| Trustant, ACP, installer plugin and skills (AGPL) | Determine and provide the corresponding source matching the actual artifacts, including relevant build/install inputs. |
| pglast (GPL), psycopg/psycopg-binary (LGPL), GPL/LGPL Ubuntu packages | Review the particular library/executable boundaries, source delivery, installation information and any replacement/relinking requirements. Merely installing an unmodified package does not prove completion of these obligations. |
| certifi, orjson, tqdm and other MPL-covered files | Identify the covered files and provide the applicable source availability notices for the distributed versions. |
| 7-Zip with unRAR restriction | Review the additional restriction separately; keep it distinct from permissive or LGPL-only components. |
| Node.js, Ollama and other downloaded runtimes | Inspect their bundled libraries and original notices; the top-level license is not an exhaustive license statement for the binary archive. |
| Font families and provider imagery | Keep font licenses separate from software licensing, and verify artwork ownership and trademark permissions independently. |

Primary texts consulted:
[AGPL-3.0-or-later](https://spdx.org/licenses/AGPL-3.0-or-later.html),
[LGPL-3.0-only](https://spdx.org/licenses/LGPL-3.0-only.html),
[Apache-2.0 section 4](https://www.apache.org/licenses/LICENSE-2.0),
[7-Zip 24.07](https://raw.githubusercontent.com/ip7z/7zip/24.07/DOC/License.txt),
[Work Sans](https://raw.githubusercontent.com/google/fonts/main/ofl/worksans/OFL.txt),
[JetBrains Mono](https://raw.githubusercontent.com/google/fonts/main/ofl/jetbrainsmono/OFL.txt),
and [pglast 7.21 metadata](https://pypi.org/pypi/pglast/7.21/json).
The component files in NOTICE are the evidence for component-specific notices;
the generic license texts are not a general compliance determination.

## Open coverage

No container digest, resolved image package list or complete transitive image
SBOM is asserted by this review. The following remain unresolved:

- Exact Ubuntu package versions, inherited base-image components and indirect
  apt dependencies. DEPS's Ubuntu expressions are declarations, not verified
  replacements for the installed `/usr/share/doc/<package>/copyright` files.
- Floating Node.js, uv, ops, global npm and Python-tool versions and their
  transitive dependency trees. Inspect both Linux architectures and the actual
  artifacts retained in every final image layer.
- Extra libraries in downloaded binaries, including Ollama's shared libraries
  and platform-specific agent packages. Their complete redistribution notices
  and corresponding source have not been established here.
- Evidence of source delivery or a source offer covering the exact image. A
  homepage link or the existence of `apt-get source` alone is not proof of it.
- Whether every production lockfile dependency contributes to the bundled ACP
  output. A bundle manifest is needed to remove unused entries safely.
- The pglast aggregate entry collects libpg_query's license; its GPL declaration
  is also documented in PyPI metadata. A complete release-level attribution
  check for pglast and all statically bundled libraries remains open.
- Upstream provenance and permission for raster artwork and inline SVG icons.
  Repository-wide AGPL headers do not establish ownership of third-party logos.
  `web/js/xterm.css` also carries the repository's header before the upstream
  MIT notice; do not treat that as evidence of relicensing upstream code.

## Artwork without per-file provenance

The following tracked assets have no per-file author/source/license declaration
in the registry. Their repository revision identifies the files under review;
it does not prove that they are third-party works or that permission is absent.
Their status is **ownership and source unverified**. Confirm first-party authorship
or record the original source and permission before using them as independently
cleared assets. Inline SVG icons in `web/*.html` need the same provenance review.

| Asset | SHA-256 | Status |
|---|---|---|
| `intro.png` | `c643f370f5936fce4f975107ded698ec6fa8ded2d7ba95c17eda0a05c0d123ba` | Unverified provenance |
| `spec/action-deploy-guard-flow.svg` | `7743f9ade18be5e39e2742ddb940dc594ef7abd6343b077f1d6bb226d1bd7235` | Unverified provenance |
| `spec/application-test-completion-gate.svg` | `1ef034cbb132702ce1291dbf546ec68b1928a1affed3660a982a3a3bae07a1e7` | Unverified provenance |
| `spec/git-pull-flow.svg` | `e565c6ced55777c900e0705d05e965b78aa2062ff9ce22fb055d3f9f116c270e` | Unverified provenance |
| `spec/issue98-action-workflow-e2e.svg` | `d5b764b7a6ae70f834607b31684710e2d316feee2b2e65a00bb8e5357db19331` | Unverified provenance |
| `spec/issue98-auth-e2e.svg` | `65d5f430bac0ebbcd388bfbc798c9bd046eaff668593333924846fecfc7d49ed` | Unverified provenance |
| `spec/issue98-compaction-e2e.svg` | `ced15b31cfbb0651218fa597280bc7ed6096c35d7aea53815a59a1ad2c510515` | Unverified provenance |
| `spec/issue98-e2e-flow.svg` | `bccd9cea58605fe0f18471b017a6c2343612e657d85f77d886d8b025598a146e` | Unverified provenance |
| `spec/issue98-session-enforcement-flow.svg` | `36e0db5b8d05ff69209342a05865d9ab1d3060973e30c01799feaa48cbd8e4ff` | Unverified provenance |
| `spec/opencode-guardrail-flow.svg` | `11ccf73c322687c1835a779ac475b3e27a630f6261fb53e29c78b9fe9696d864` | Unverified provenance |
| `spec/provider-e2e-flow.svg` | `9a513e66e862572c235b55809654dabab9553fa497bd709ea02758289fe9fb3c` | Unverified provenance |
| `spec/trustant-auth-flow.svg` | `06cf3d6a5ade7c9d69e8018f3137344ac344dc358e44426b0b5b900b2e341f46` | Unverified provenance |
| `spec/trustant-code-integration.svg` | `82ec9ac5cefd329ecb1994147da56cc2314b754ae473fa4e7c66705584ef22f4` | Unverified provenance |
| `splash.png` | `a780538a8a9fa45b29dae0d0aa06763b41a8df75b23c8e6cfb823da819f0cd9b` | Unverified provenance |
| `web/favicon.ico` | `5cc2560eddb2d9f841e425f3dfc701a47b0fedf3d2b969730bcead80bfee06e6` | Unverified provenance |
| `web/logo-trustant.png` | `6843d0682cbfb3deee6439bdb8211adb24b2a61b5b7e82a5c4eb7643c77a0540` | Unverified provenance |
| `web/ollama-head.png` | `c1f0bce3a7363d8030142943098889466f5bb21d67759963ece662a73ab6f105` | Unverified provenance |
| `web/privateai-head.png` | `8e428a3e82ddee0e13b0f0ed7e008f81ec5bf50eb93294b2ab8b0f06bfd57411` | Unverified provenance |
| `web/regolo-head.png` | `dc654fa1a66ee1919d4656b67723c81dd80bc65bbebb09001a2d6cdb8a7efadb` | Unverified provenance |
| `web/trustant-head.png` | `6843d0682cbfb3deee6439bdb8211adb24b2a61b5b7e82a5c4eb7643c77a0540` | Unverified provenance |
| `web/trustant-logo-small.png` | `e5ad92afcef328909c0cd5be8af0b388bd7616dbba25e946b1a54202aba13368` | Unverified provenance |
| `web/trustant-logo.png` | `bfdde3b38d2eaeac1dcc63db41abdd5d411de1d2b82097241df1c4c215d42bf4` | Unverified provenance |
| `web/trustant-logo.svg` | `4634c3087e71f577325131fab2dd4faa285f1d6a679d66bd563b982c93a4990b` | Unverified provenance |
| `web/trustant.png` | `6843d0682cbfb3deee6439bdb8211adb24b2a61b5b7e82a5c4eb7643c77a0540` | Unverified provenance |
| `web/wa-trustant.jpeg` | `288742cacea2d6b447e6c1b6644750c5c00add6a283e258ca1977ba8b31b70d5` | Unverified provenance |
