---
name: ci-failure-triage
description: Triages a failing CI run from its logs. Use when a pull request or push shows failing checks.
---

# CI failure triage

1. List the failing jobs and open the first failing step's log.
2. Separate the root failure from the follow-on errors.
3. Reproduce locally with the same command when possible.
4. Report the cause, the minimal fix and anything you could not verify.
5. Do not re-run or disable checks to make them pass.
