# trustant-acp - LLM Developer Guide

## Overview
`trustant-acp` is a standalone ACP client (local Node server + browser React UI) for AI agent interaction (Pi, Claude Code, Codex, custom agents) via the Agent Client Protocol. See [SPEC.md](SPEC.md) for the full specification — it is the source of truth.

**Tech**: React 18, TypeScript, Node server, Agent Client Protocol (ACP)

**Repository**: this directory is vendored in the Trustant repository (including `pi-acp/`); it is not a submodule and has no separate remote. Upstream is not tracked or merged; do not assume upstream updates or propose syncing from it unless asked. Commits and pushes follow the root [AGENTS.md](../AGENTS.md).

**Licensing**: this directory is vendored and partly derived from obsidian-agent-client (Apache-2.0, Copyright RAIT-09). Follow the third-party license and vendored-source rules in the root [AGENTS.md](../AGENTS.md); upstream attribution is in [UPSTREAM.md](UPSTREAM.md) and [NOTICE](NOTICE). Do not add AGPL headers to `pi-acp/` files (MIT).

## Architecture

- `src/acp/` handles ACP process lifecycle, protocol events, permissions and terminals.
- `src/types/`, `src/utils/` and `src/services/notebook.ts` provide shared types, helpers and notebook data.
- `server/` owns process execution, filesystem access, session persistence and HTTP/WebSocket transport.
- `web/` is the standalone browser client. `ChatApp.tsx` owns the conversation and notebook UI.
- `pi-acp/` is the separately maintained vendored agent. Follow its local directives for changes there.

## Development

- Keep filesystem operations, subprocess management and secrets on the server.
- Keep ACP SDK protocol handling in the portable core and the server.
- Use React for DOM rendering, CSS for styles, and specific types for data crossing the transport boundary.
- For behavioral changes, add or update a nearby comment explaining the runtime or compatibility rationale. Skip comments for purely mechanical edits.
- Run `npm test` and `npm run build` from this directory and `npm run lint` from the repository root.

See [ARCHITECTURE.md](ARCHITECTURE.md) for the component map and [SPEC.md](SPEC.md) for the runtime contracts.
