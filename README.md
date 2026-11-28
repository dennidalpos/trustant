# Trustant

Development environment for creating, editing, running and publishing web applications backed by OpenServerless.

## Purpose

Trustant combines a browser UI, the TruACP/Pi coding assistant, Git repositories for application source and local development services. It runs as a server used from a web browser; the repository has no desktop application. [Demo](https://trustant.ai/#video?v=demo) · [Website](https://trustant.ai)

## Requirements

- Git, and a host that can run the Ubuntu toolchain: Windows with WSL2, macOS with Lima, or Ubuntu 24.04 (amd64 or arm64) with systemd and a user with passwordless sudo.
- Network access: setup downloads the pinned serverless package and dependencies.

## First setup

```powershell
git clone https://github.com/trustant/trustant.git
cd trustant
.\start.ps1      # Windows: creates the WSL2 distribution `trudev`
```

On macOS (`brew install lima` first) and on Ubuntu, run `./start.sh` instead. The script prepares the host, runs `setup.sh` and starts the development loop (`run.sh`). Open the Trustant URL it prints, configure a provider and model in the UI, then create or import an application. Startup creates `.env` if absent. Step-by-step guides: [Windows](docs/setup-windows.md), [macOS](docs/setup-macos.md), [Ubuntu](docs/setup-ubuntu.md).

## Usage

Use the app list to create or import an application, then work in its workbench: assistant, preview, file viewer, terminal and configuration. Save source with the Git controls. Publishing to a remote production destination and provisioning need a valid license; Git push and development deploys do not. See [use cases](docs/use-cases.md) and [how it works](docs/how-it-works.md).

The `-k` option of the start scripts destroys the VM or unregisters the WSL distribution and its filesystem: use it only to discard that environment.

## Build and test

### Local checks

Run Go checks in the prepared Ubuntu environment (the root package does not compile on Windows):

```bash
if [ ! -f _build.txt ]; then
  printf 'Version: %s\nBuild: local-test\nBranch: development\nStream: development\n' "$(cat version.txt)" > _build.txt
fi
go test ./...
npm run test:tutorial
```

The TruACP suites (`npm test` in `acp/` and `acp/pi-acp/`) also assume POSIX paths. Release tooling is in [spec/build.md](spec/build.md): `build.sh --tag` and `--build` can replace local tags and commit.

## Documentation and security

[Docs index](docs/README.md) · [Specifications](spec/) · [Full previous README](REFERENCE.md) (configuration, hostnames, publishing, troubleshooting) · [Contributing](CONTRIBUTING.md) · [Agent instructions](AGENTS.md) · [Security policy](SECURITY.md) · [License](LICENSE), [NOTICE](NOTICE), [DEPS](DEPS) and [CREDITS.md](CREDITS.md).
