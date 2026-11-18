# Security Policy

## Supported versions

trustant-acp is developed from `main` and ships inside the Trustant image.
Security fixes are made on `main`; older builds are not patched separately.

## Reporting a vulnerability

Do not open a public issue for a security problem.

Use GitHub's private reporting: open the repository's **Security** tab and
choose **Report a vulnerability**. If that option is not available, contact
**info@nuvolaris.io** and say the message is a security report.

Include the affected version or commit, the host platform, steps to reproduce, and the impact. Remove credentials,
tokens, and private data from logs before sending them.

## Scope

In scope: the Node server and its REST and WebSocket interface, the browser UI,
secret and configuration handling, agent launching, the managed runtime
extension, and the setup scripts in this repository.

Out of scope: vulnerabilities in third-party dependencies that are not
reachable from Trustant (report them upstream), and issues that require
an already compromised development machine.

## Handling

Reports are acknowledged, reproduced, and fixed on `main`. Please keep the
details private until a fix is available. Secrets must never be committed; see
[AGENTS.md](AGENTS.md) for how credentials are kept in their authentication
stores.
