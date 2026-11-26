# Credits

Trustant is built on the work of many projects. This file credits the main
components and says where each comes from and under which license. It is a
readable summary; the complete, generated registers are [DEPS](DEPS),
[LICENSE](LICENSE), [NOTICE](NOTICE), and [legal/COMPONENTS.json](legal/COMPONENTS.json),
produced by `./deps.sh` (see [spec/20-legal.md](spec/20-legal.md)). When this
file and the generated registers disagree, the generated registers are right:
fix this file.

Trustant itself is AGPL-3.0-or-later, Copyright 2025-2026 Nuvolaris Inc. See
[CONTRIBUTING.md](CONTRIBUTING.md) for the contributor agreement.

## Sources in this repository

| Component | Role | License | Origin |
|---|---|---|---|
| `acp/` (TruACP) | Assistant runtime: Node server and browser UI over the Agent Client Protocol | AGPL-3.0-or-later for Trustant's parts; Apache-2.0 for the parts derived from obsidian-agent-client | Vendored from the owner's `trustant-acp` fork, itself derived from obsidian-agent-client (below) |
| obsidian-agent-client 0.11.0 | Origin of `acp/`; derived files carry its attribution | Apache-2.0, Copyright 2025-2026 RAIT-09 | https://github.com/RAIT-09/obsidian-agent-client |
| `acp/pi-acp/` | Adapter between ACP and the Pi coding agent | MIT, Copyright (c) 2025 Sergii Kozak | Vendored fork of `svkozak/pi-acp` 0.0.31 (`trustant/pi-acp`) |
| `web/` assets | Browser UI libraries | MIT | See below |

Vendored web assets: [Tailwind CSS browser build](https://github.com/tailwindlabs/tailwindcss),
[marked](https://github.com/markedjs/marked), [CodeJar](https://github.com/antonmedv/codejar),
[xterm.js](https://github.com/xtermjs/xterm.js) and the [Heroicons](https://github.com/tailwindlabs/heroicons)
v1 outline icons (MIT, Copyright (c) 2020 Refactoring UI Inc.), inlined as SVG in
`web/app.html` and `web/applist.html`; some were adapted. Fonts Work Sans and
JetBrains Mono (SIL OFL-1.1) are loaded from Google Fonts at runtime and are not
bundled.

## Submodules

| Component | License | Source |
|---|---|---|
| `mcp` (OpenServerless MCP) | Apache-2.0 | https://github.com/trustant/openserverless-mcp |
| `oplugins` (OpenServerless tasks) | Apache-2.0 | https://github.com/trustable-ai/openserverless-task |
| `oplugins-truinst` | AGPL-3.0-or-later | https://github.com/trustant/oplugins-truinst |
| `skills` | Apache-2.0 | https://github.com/trustable-ai/skills |

The Apache-2.0 components (`mcp`, `oplugins`, `skills`) are vendored in this repository (formerly gitlinks; per-file changes in [legal/VENDORED.md](legal/VENDORED.md)) and keep their own LICENSE and
NOTICE files and carry no AGPL headers.

For vendored sources, upstream attribution is in [acp/UPSTREAM.md](acp/UPSTREAM.md)
and `acp/NOTICE`, and per-file changes are tracked in
[legal/VENDORED.md](legal/VENDORED.md).

## Coding agents and protocol

- [Pi](https://github.com/earendil-works/pi) packages (`pi-ai`, `pi-tui`,
  `pi-agent-core`, `pi-coding-agent`), MIT.
- [Agent Client Protocol](https://agentclientprotocol.com) SDK and the Codex ACP
  adapter, Apache-2.0.
- [Zod](https://zod.dev/) schema validation for the ACP and React MCP SDKs, MIT.
- [OpenAI Codex CLI](https://github.com/openai/codex), Apache-2.0.
- Claude Code is not distributed with Trustant; TruACP installs it on demand
  after the user accepts Anthropic's terms.

## Platform and tools

Trustant runs on [OpenServerless](https://openserverless.apache.org)
(Apache-2.0) over k3s, and its image and setup use, among others: Node.js (MIT),
Go (BSD-3-Clause), Ollama (MIT), GitHub CLI (MIT), uv (Apache-2.0 or MIT),
kubectl, kind and Helm (Apache-2.0), bun, yq, k3sup and uutils coreutils (MIT), and
Ubuntu packages under their distribution licenses. Python application packages
and npm production trees are listed in full in [DEPS](DEPS).

The vendored `oplugins` sources include the Apache OpenServerless logo
(`oplugins/setup/openserverless/web/os-logo-full-vertical-bg-white.png`). It is a
mark of that project, not of Trustant; the Apache-2.0 license of the sources does
not grant rights over it (see [legal/REVIEW.md](legal/REVIEW.md)).

The provider logos `web/ollama-head.png`, `web/regolo-head.png` and
`web/privateai-head.png` are marks of Ollama, Regolo.AI and Private AI. They are
shown only to identify those options in the UI; this project's license grants no
rights over them. Inline SVG icons that are not byte-identical to Heroicons are first-party or
adapted from Heroicons, as confirmed by the owner (see [legal/REVIEW.md](legal/REVIEW.md)).

The image builds `ops` from the [Trustant OpenServerless CLI fork](https://github.com/trustable-ai/openserverless-cli)
with [go-git](https://github.com/go-git/go-git) (both Apache-2.0).
The updated Go modules are listed in [DEPS](DEPS). The unchanged
filepath-securejoin source is available from the
[v0.6.1 release](https://github.com/cyphar/filepath-securejoin/tree/v0.6.1)
under BSD-3-Clause and MPL-2.0.

The setup and image use [Redis MCP](https://github.com/redis/mcp-redis) and
[tsx](https://github.com/privatenumber/tsx) under MIT. The image also includes
[TypeScript](https://github.com/microsoft/TypeScript) under Apache-2.0 and the
[TypeScript language server](https://github.com/typescript-language-server/typescript-language-server)
under Apache-2.0 with MIT-licensed portions from VS Code.

Browser end-to-end tests use [Playwright](https://github.com/microsoft/playwright)
(Apache-2.0).

TruACP development uses the Node.js declarations from
[DefinitelyTyped](https://github.com/DefinitelyTyped/DefinitelyTyped/tree/master/types/node)
and their transitive [Undici](https://github.com/nodejs/undici) declarations (MIT).

Development setup uses [kubefwd](https://github.com/txn2/kubefwd) and
[SkyWalking Eyes](https://github.com/apache/skywalking-eyes) (Apache-2.0).

Development lint uses [Biome](https://biomejs.dev) (MIT or Apache-2.0),
[Ruff](https://github.com/astral-sh/ruff) (MIT),
[ShellCheck](https://github.com/koalaman/shellcheck) and
[Hadolint](https://github.com/hadolint/hadolint) (GPL-3.0-only), and
[PSScriptAnalyzer](https://github.com/PowerShell/PSScriptAnalyzer) (MIT).

## License compatibility notes

- AGPL-3.0-or-later code may include Apache-2.0, MIT, BSD, and similar
  permissive code; their notices are preserved in [NOTICE](NOTICE).
- Tools licensed GPL-2.0-only or GPL-3.0 (for example `git`, or `pglast` inside
  the Postgres MCP server) run as separate programs or tools and are not linked
  into Trustant. They are listed for attribution, not combined into one work.
- `7zz` carries an unRAR restriction; it is downloaded by the OpenServerless
  prerequisites and is not redistributed in the source tree.
- Items marked `declared-only` in the registers have license metadata but no
  collected license text. They are not legal clearance. See
  [legal/REVIEW.md](legal/REVIEW.md).
- The `trustable-ai` GitHub organization is the owner's fork namespace and is
  intentional, not a leftover of the rebrand (owner decision 2026-11-25, task
  #30): `MILVUS_MCP_REPO` (`trustable-ai/mcp-server-milvus`) and the default
  `ops` task source (`trustable-ai/openserverless-task`) in
  [image/Dockerfile](image/Dockerfile), the `openserverless-cli` tarball
  and the default notebook templates `trustable-ai/templates`. Rename them only
  when the owner moves those repositories.

- Source code for the GPL/LGPL components of the container image is offered in
  [legal/SOURCE-OFFER.md](legal/SOURCE-OFFER.md).

## Keeping this file current

Update this file in the same change that adds, removes, or re-sources a
component, then rerun `./deps.sh` and commit the regenerated registers.
