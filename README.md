# Trustant

Trustant is a development environment for creating, editing, running, and publishing web applications backed by OpenServerless. It combines a browser UI, the TruACP/Pi coding assistant, Git repositories for application source, and local development services on k3s.

This guide helps new users run Trustant from source and create or import an application. Contributors can also use it to find the development and testing guides.

Trustant runs as a server and is used from a web browser. The repository has no desktop application.

[Watch the Trustant demo](https://trustant.ai/#video?v=demo) · [Website](https://trustant.ai)

## Documentation

| I want to | Read |
|---|---|
| Install and start Trustant | [Windows](docs/setup-windows.md), [macOS](docs/setup-macos.md) or [Ubuntu](docs/setup-ubuntu.md) |
| Understand the architecture | [How it works](docs/how-it-works.md) |
| Do something with it | [Use cases](docs/use-cases.md): first run, building an app, a local model with Ollama, publishing |
| Know the exact intended behavior | [spec/](spec/) |
| Contribute | [CONTRIBUTING.md](CONTRIBUTING.md) and [AGENTS.md](AGENTS.md) |

## How it fits together

```mermaid
flowchart LR
    Browser[Browser UI<br/>web/] -->|trustant.DOMAIN| Go[Go server :8910<br/>main.go, middleware.go]
    Browser -->|truacp.DOMAIN| ACP[TruACP assistant :4096<br/>acp/]
    Browser -->|vite.DOMAIN| Vite[Vite preview :5173]
    Go --> WS[(WORKSPACE_DIR<br/>bare repos + trustant.json)]
    Go --> WB[(WORKBENCH_DIR<br/>active checkouts)]
    ACP -->|spawns| Pi[Pi / Codex / Claude agents]
    ACP --> WB
    Go -->|deploy, provision| OS[OpenServerless on k3s]
```

The Go server owns configuration, repositories, launch, publishing, and the license checks. The assistant runs as a separate process reached through its own hostname; it works on the same application checkout the preview serves. The hostname routing and its authentication boundary are diagrammed in [request routing](spec/authentication.md#request-routing).

## Quick start

Use the entry point for your host. The scripts install tools and services, configure networking, and start the development server; first startup downloads the pinned serverless package and runtime dependencies. Keep the checkout and its generated configuration on a development machine.

### Windows (PowerShell and WSL2)

Install Git and ensure WSL2 is available (`wsl --version`). The script creates a dedicated `trudev` distribution from Ubuntu 24.04, configures systemd and a user with passwordless sudo, and runs the Linux setup inside it.

```powershell
git clone https://github.com/trustant/trustant.git
cd trustant
.\start.ps1
```

The source stays in the Windows checkout and is accessed through the WSL mount. Follow the printed instructions to enter the distribution or open the UI. See [Windows setup and options](spec/start.md).

### macOS (Lima)

Install Git and Lima. With Homebrew available:

```bash
brew install lima
git clone https://github.com/trustant/trustant.git
cd trustant
./start.sh
```

The script provisions the `trudev` Ubuntu VM, mounts the checkout, runs setup, and starts the development loop inside the VM. The Lima configuration allocates 8 GiB of memory and 60 GiB of disk; allow additional host capacity for the checkout and download cache. See [startup details](spec/start.md).

### Native Ubuntu

Use Ubuntu 24.04 on amd64 or arm64 with systemd and a user with passwordless sudo. Provisioning installs the local serverless cluster on this host.

```bash
git clone https://github.com/trustant/trustant.git
cd trustant
./start.sh
```

The script prepares the host, runs `setup.sh`, then starts `run.sh`. See [native Ubuntu setup](spec/start.md#native-ubuntu-linux).

Step-by-step guides per system, with diagrams, are in [docs/](docs/README.md): [Windows](docs/setup-windows.md), [macOS](docs/setup-macos.md), [Ubuntu](docs/setup-ubuntu.md), plus [how it works](docs/how-it-works.md) and [use cases](docs/use-cases.md).

### What the start scripts do

```mermaid
flowchart TD
    A[start.ps1 / start.sh] --> B{Host}
    B -->|Windows| C[Create WSL2 distro trudev]
    B -->|macOS| D[Boot Lima VM trudev]
    B -->|Ubuntu| E[Use this host]
    C --> F[setup.sh]
    D --> F
    E --> F
    F --> G[run.sh: dev loop with Air]
    G --> H[Go server + TruACP + Vite]
    H --> I[Open the printed Trustant URL]
```

### Open the UI

1. Open the URL printed by the development loop in your browser.
2. Configure a provider and its credentials in the UI, then choose a model. Use **Private AI** to [connect an OpenAI-compatible endpoint](docs/use-cases.md#use-an-openai-compatible-endpoint), or follow the [local Ollama walkthrough](docs/use-cases.md#4-use-a-local-model-ollama).
3. Create or import an application from the app list and open it.

The app workbench provides the assistant, preview, file viewer, terminal, and application configuration.

`mcp/`, `oplugins/`, `skills/` and TruACP (`acp/`, including `acp/pi-acp/`) and `oplugins-truinst/` are part of this repository; there are no submodules to initialize.

Startup creates `.env` when it is absent; existing values are validated by setup. See [configuration](#configuration) before customizing it.

## Common tasks

### Restart development or open VS Code

| Task | macOS | Windows PowerShell | Native Ubuntu |
|---|---|---|---|
| Start environment and development loop | `./start.sh` | `.\start.ps1` | `./start.sh` |
| Prepare without starting the loop | `./start.sh -n` | `.\start.ps1 -n` | `./start.sh -n` |
| Open VS Code in the environment | `./start.sh -v` | `.\start.ps1 -v` | Open the local checkout |
| Stop the VM or distribution | `./start.sh -s` | `.\start.ps1 -s` | Not applicable |

On Windows, `.\start.ps1 -NoStart` only provisions the distribution without running `./start.sh`, and `-Distro <name>` selects another WSL instance name (default `trudev`).

Run `./run.sh` from the checkout **inside Ubuntu** to restart only the development loop. On macOS, `./ssh.sh ./run.sh` runs it in the VM. Press Ctrl+C to stop the loop. `run.sh` rejects unsupported hosts and frees ports 8910, 5173, and 4096 before starting services.

`./ssh.sh -d` opens a development shell in the macOS VM. The `-p` and `-i` modes access the installed Trustant user and the deployed container; see [SSH access](spec/start.md#sshsh-has-three-access-modes).

The `-k` startup option destroys the VM or unregisters the WSL distribution and removes its filesystem. Use it only when you intend to discard that environment. Native Linux refuses the VM stop and destroy options.

### Update an existing checkout

Stop the development loop and review local changes before updating. In the development environment, pull the branch you use, then refresh setup:

```bash
git status --short
git pull --ff-only
./setup.sh
./run.sh
```

`setup.sh` installs and verifies the toolchain and MCP servers in Ubuntu. Its runtime pins come from the repository and [acp/pi.version](acp/pi.version). On macOS, enter the VM with `./ssh.sh -d` first; on Windows, use the shell command printed by `start.ps1`.

### Save and publish an application

Use the app's Git controls to save source changes and push to its configured Git remote. Configure production environment values before deploying to an OpenServerless destination.

```mermaid
flowchart TD
    P[Publish request] --> Q{Operation}
    Q -->|Git push / force-push| OK1[Allowed, no license check]
    Q -->|Development deploy| OK2[Allowed, no license check]
    Q -->|Remote production deploy| L{Valid license?}
    Q -->|Provisioning| L
    L -->|No| E1[License required]
    L -->|Yes| H{Destination host licensed?}
    H -->|local hostname| OK3[Proceed]
    H -->|licensed| OK3
    H -->|not licensed| E2[Host not licensed]
```

With license enforcement enabled (the default):

| Operation | License checks |
|---|---|
| Git push or force-push (`/api/publish/push`, `/api/publish/force-push`) | None |
| Development deployment (`/api/git/deploy`) | None |
| Remote production deployment (`/api/publish/remote`) | Valid license and destination host |
| Application provisioning (`/api/publish/provision`) | Valid license and configured license host |

Install a license and configure provisioning settings on Configure. Verification uses an embedded Ed25519 public key. Destination matching compares normalized `scheme://host[:port]` values without wildcards. The literal local hostnames `miniops.me`, `localhost`, `127.0.0.1`, and `::1` bypass only the host check; production operations still require a valid license.

The backend returns license errors with the `License required` or `Host not licensed` prefix, which the UI uses to open its license dialog. See [publishing](spec/6-publish.md), [provisioning](spec/user_provisioning.md), and [license behavior](spec/14-license.md). License issuance is maintained in the separate installer repository.

## Configuration

### Server environment

The Go server loads a mandatory `.env` from its current directory. Startup seeds it from [.env.dist](.env.dist); [setup.sh](setup.sh) validates existing values. The template is the starting point for manual setup, not a provider credential file.

| Variable | Purpose |
|---|---|
| `WORKSPACE_DIR` | Durable configuration and bare application repositories under `workspace/<name>/` |
| `WORKBENCH_DIR` | Active application checkouts under `<name>/` |
| `OLLAMA_ENDPOINT` | Ollama service endpoint |
| `AIP_REGISTER_URL` | Registration UI base URL |
| `AIP_BASE_URL` | Credits and top-up API base URL |
| `GIT_USER`, `GIT_EMAIL` | Git identity defaults validated by setup; app configuration also carries Git identity |
| `GITHUB_TOKEN` | Optional helper credential; the template placeholder is not a working token |

The first five variables are required by the server's `loadEnv` function in [preflight.go](preflight.go). `OPENAI_BASE_URL` and `OPENAI_API_KEY` are not mandatory server variables; the selected provider is stored in workspace configuration and credentials are propagated to the assistant runtime.

An optional `.env.default` seeds the predefined environment variable palette in Configure. It is separate from server `.env` and per-app environment files. See [configuration behavior](spec/2a-config.md).

### Application configuration and storage

Trustant merges the defaults in root [trustant.json](trustant.json) with user configuration in `$WORKSPACE_DIR/trustant.json`. Provider settings, selected model, applications, and license settings persist in the workspace layer. Each app has separate development and production environment maps.

- `$WORKSPACE_DIR/workspace/<name>/` is the durable **bare Git repository**.
- `$WORKBENCH_DIR/<name>/` is its **active checkout**. Launch restores missing checkouts and regenerates managed configuration, including environment files.

Save source changes through the app's Git workflow. Keep credentials out of source repositories. See [repository management](spec/2-repo.md), [launch lifecycle](spec/4-launch.md), [managed files](spec/13-gitignore.md), and [Pi runtime](spec/trustant-pi-runtime.md).

### Hostnames and access

The Go listener on `:8910` dispatches by the first hostname label, as implemented in [middleware.go](middleware.go):

| Host | Destination |
|---|---|
| `trustant.<domain>` | Trustant UI and Go API |
| `truacp.<domain>` | TruACP assistant on `127.0.0.1:4096` |
| `vite.<domain>` | Vite preview on `127.0.0.1:5173` |
| Registered application name | That application's development service |
| Unknown label | HTTP 400 with a suggested Trustant URL |

```mermaid
flowchart LR
    R[Request on :8910] --> M{First host label}
    M -->|trustant| U[UI and Go API]
    M -->|truacp| T[127.0.0.1:4096]
    M -->|vite| V[127.0.0.1:5173]
    M -->|registered app name| AP[App dev service]
    M -->|other| X[HTTP 400 + suggested URL]
    R -.->|localhost, IP, no dot| RD[HTTP 307 to Trustant FQDN]
```

Requests using `localhost`, an IPv4 address, or a hostname without a dot receive an HTTP 307 redirect to the Trustant FQDN. Use the printed FQDN for browser access and hostname-routing checks. These surfaces use separate hostnames.

Local administrator authentication is optional and disabled by default; the server logs a warning at startup while it is off, and port 8910 uses plain HTTP, so use it on a trusted network. See [authentication scope and configuration](spec/authentication.md) before exposing the environment outside your development machine.

## Contributing and testing

Read [AGENTS.md](AGENTS.md) for repository instructions and [CONTRIBUTING.md](CONTRIBUTING.md) for contribution requirements, including the contributor agreement. Intended behavior is described under [spec/](spec/); flag disagreements with the implementation when making changes.

The backend uses one root Go package with feature files such as [launch.go](launch.go), [configure.go](configure.go), and [publish.go](publish.go). API routes are registered in [main.go](main.go); refer there for the current route inventory. Responses include JSON, progress event streams, and terminal WebSockets.

The `web/` UI uses HTML and JavaScript with Tailwind. During development the server serves it from disk, so browser reloads pick up frontend edits. Air watches root Go source and version text; see [.air.toml](.air.toml). Production builds embed UI assets and app instructions, while still relying on the external runtime tools and services.

### Local checks

Run Go checks in the prepared Ubuntu environment; the root package does not compile on Windows (it uses `syscall.Kill` and `Setpgid`). `go.mod` declares the required Go version. A clean checkout lacks `_build.txt`, which `main.go` embeds. `run.sh` generates it before starting Air; to run tests without starting services, create local test metadata first:

```bash
if [ ! -f _build.txt ]; then
  printf 'Version: %s\nBuild: local-test\nBranch: development\nStream: development\n' "$(cat version.txt)" > _build.txt
fi
go test ./...
```

The file is Git-ignored. Existing metadata is preserved by this example.

On Windows, open the checkout in an editor connected to the prepared WSL
distribution so the Go language server uses the Linux toolchain. In an
existing checkout with local Go files under `tmp/` or `node_modules/`, use
`go vet .` and `go test .` from the repository root to check the Trustant
package. Use `go test ./...` in a clean checkout as CI does.

The lightweight Node checks use the built-in test runner:

```bash
npm run test:tutorial
```

The TruACP suites (`npm test` in `acp/` and `acp/pi-acp/`) assume POSIX paths and processes. Run these suites in the Ubuntu environment; they are not reliable on a native Windows host.

The OpenCode cluster E2E scenarios were removed with the runtime; see the [E2E guide](spec/8-e2e.md).

### Submodules and releases

The repository has no submodules: `mcp`, `oplugins`, `skills` and `oplugins-truinst` are vendored directories (see [legal/VENDORED.md](legal/VENDORED.md)). The TruACP runtime lives in [acp/](acp/), with its Pi ACP adapter in `acp/pi-acp/`; follow [acp/AGENTS.md](acp/AGENTS.md) there.

Release tooling is documented in [spec/build.md](spec/build.md). `build.sh --tag` and `--build` can replace local tags, update `oplugins-truinst/opsroot.json`, and attempt a commit, including with `--no-deploy`. `hotfix.sh` can replace tags and patch the live StatefulSet; a regular redeploy restores the image from `opsroot.json`. Buildx modes push images, and `publish.sh` pushes Git refs. Review the script before using a release command.

[openserverless-instructions.md](openserverless-instructions.md), [openserverless-contract.md](openserverless-contract.md), and [app-agents.md](app-agents.md) are product assets supplied to generated applications. Repository development instructions live in [AGENTS.md](AGENTS.md).

## Limits and troubleshooting

- The supported source-development path runs the toolchain on Ubuntu, directly or through Lima/WSL2. Running the Go server alone does not provision the cluster, assistant, or MCP services.
- Setup needs network access for pinned packages and dependencies. Model inference, Git hosting, and remote deployments depend on their configured services.
- On native Linux, a serverless package version mismatch can prompt to purge and reinstall the cluster package, removing cluster data. Read the prompt before accepting; see [startup behavior](spec/start.md).
- If compilation reports a missing `_build.txt`, use the test metadata step above or the normal `run.sh` development loop.
- If a hostname redirects unexpectedly or returns 400, use the FQDN printed by startup and check the routing table above.
- For setup, runtime, or deployment failures, consult [startup](spec/start.md), [setup](spec/setup.md), [development loop](spec/run.md), and [publishing](spec/6-publish.md).

This README's commands were reviewed against the scripts. Full provisioning, model inference, and publication still require verification on the target host and configured services.

## Help and license

Report reproducible problems in the [Trustant issue tracker](https://github.com/trustant/trustant/issues), including the relevant version, host platform, and sanitized error output. For contribution questions, follow [CONTRIBUTING.md](CONTRIBUTING.md). Report security problems privately as described in [SECURITY.md](SECURITY.md).

See [LICENSE](LICENSE), [NOTICE](NOTICE), and [DEPS](DEPS) for licensing and third-party notices, and [CREDITS.md](CREDITS.md) for a readable summary of the components Trustant builds on.
