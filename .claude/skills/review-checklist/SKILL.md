---
name: review-checklist
description: Reviews a diff against a short checklist. Use when asked to review a change, a branch or a pull request.
---

# Review checklist

- Correctness: does the diff do what the request says, including edge cases and error paths?
- Scope: does it change anything the request did not ask for?
- Tests: is new behavior covered, and were the relevant checks run?
- Safety: secrets, paths, input from outside the program, destructive operations.
- Report findings ordered by severity, each with the file and line, and say what you did not check.
