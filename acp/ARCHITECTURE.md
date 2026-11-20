# TruACP architecture

TruACP runs as a Node server and a browser client. The server runs agent processes and sends session events over WebSocket. The browser uses HTTP for requests and renders sessions, notebooks and prompts in React.

| Directory | Responsibility |
|---|---|
| `src/acp/` | ACP client, event conversion, permissions and terminal lifecycle |
| `src/types/`, `src/utils/` | Shared types and platform utilities |
| `src/services/notebook.ts` | Notebook model and shared data operations |
| `server/` | Agent sessions, configuration, local persistence, HTTP and WebSocket |
| `web/` | Browser transport, conversation, composer and notebook UI |
| `pi-acp/` | Vendored Pi agent adapter |

The browser build embeds its assets in the Node server bundle. See [SPEC.md](SPEC.md) for behavior and [UPSTREAM.md](UPSTREAM.md) for provenance and attribution.
