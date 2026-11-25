---
name: project-checks
description: Runs the right project checks before work is reported as done. Use when finishing a code change or when asked to verify, test or lint the project.
---

# Run the project checks

Before you report a change as done, run the checks that fit what you touched and report their real output.

Detected commands (confirm them against the project's own instructions before relying on them):
- `npm run lint`
- `go test ./...`

- Run the narrowest check first, then the wider ones.
- If a check fails, show the failure; do not claim success.
- Say which checks you did not run.
