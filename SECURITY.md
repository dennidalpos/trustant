# Security Policy

## Supported versions

Only the latest commit on the `main` branch is supported. Security fixes are made on `main` and released through the normal image build; older builds are not patched separately.

## Reporting a vulnerability

Please do not open a public issue for security problems.

Use GitHub's private vulnerability reporting: open the **Security** tab of this repository and choose **Report a vulnerability**. If that option is not available, contact **info@nuvolaris.io** and say the message is a security report. Include the affected version or commit, the host platform (Windows/WSL2, macOS/Lima or Ubuntu), the steps to reproduce and the impact. Remove credentials, tokens and private data from logs before sending them.

You can expect an acknowledgement within a few days. Confirmed issues are fixed on `main` and disclosed after a fix is available.

## Scope notes

- In scope: the Go server and its API, the web UI, the launch, publish and license flows, the managed assistant runtime, and the setup and release scripts in this repository.
- Out of scope: vulnerabilities in third-party dependencies that are not reachable from Trustant (report them upstream), and issues that require an already compromised development machine.
- Secrets must never be committed; see [AGENTS.md](AGENTS.md) for how credentials are kept in their authentication stores.
