# Application completion checks (TruACP and Pi)

> The former OpenCode completion gate (`trustant_completion_check`, the session
> plugin, its recovery turns and raw-command guards) was removed with the
> OpenCode runtime. It is not enforced anywhere. This document specifies what
> exists today for TruACP with Pi. The previous text stays in Git history; see
> [pi.md](pi.md) "Trustant execution policy" for why Pi has no tool-call gate.

## Goal

Before an assistant reports an application change as complete, the source
contract is checked by commands the assistant runs itself. Nothing intercepts
or blocks a final answer: the checks are instructions delivered through the
managed `AGENTS.md` ([openserverless-instructions.md](openserverless-instructions.md))
and the assistant is expected to follow them. A missing check is a prompt
compliance failure, not a runtime error.

## Checker scripts

The scripts live in the repository root and are installed once in the user
`PATH`:

| Script | Checks |
| --- | --- |
| `check_openserverless_actions.sh` | OpenServerless action source contract |
| `check_trustant_frontend.sh` | Frontend contract (for example no localStorage identity as authentication) |
| `check_trustant_app.sh` | Aggregate: runs the two above, each with a 60 second timeout, and exits non-zero if any fails |

Run them as `timeout 60 <script> .` from the application root. They never
install dependencies or download code.

## Expected sequence after source changes

1. Run the relevant checker script directly; do not pipe it into `head` or
   `tail`, which hides its exit status. Reading a checker file is allowed.
2. Run `git diff --check`.
3. For frontend work, run the project `typecheck` script before the build. When
   no script exists but a local TypeScript compiler and `tsconfig.json` do, run
   `tsc -b --noEmit --incremental false`, so a Vite-only build cannot hide
   undefined JSX symbols or leave `.tsbuildinfo` artifacts.
4. Run the frontend build when available, then confirm the changed route with
   `react_validate` and bounded HTTP checks against the managed development
   server.
5. Run the application tests that already exist (for example `go test ./...`
   or the declared `test` script). Do not add placeholder tests to satisfy a
   check, and do not install a test framework only for this purpose.

For OpenServerless actions, deploy and setup order is defined in
[openserverless-instructions.md](openserverless-instructions.md). Actions are
changed only through the OpenServerless MCP and `ops ide deploy` / `ops ide
setup`; `ops ide login` is not repeated, because Trustant already
authenticated and configured the application at launch.

## Not provided

- No tool call is intercepted, so raw `ops action` / `wsk action` shell
  commands and `ops ide login` are discouraged by instructions only.
- No persisted verification debt, no per-endpoint test requirement and no
  forced recovery turn.
- A guardrail would need a reviewed Pi execution-policy extension (see
  [pi.md](pi.md)); until one exists this specification does not describe
  enforcement.
