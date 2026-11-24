This file describes the skills feature.
Put the backend code in `skills.go`.

# Initial skill setup

On every app launch:
- if `.agents` does not exist, create it
- if `.agents` exists and is not empty, skip (skills already installed)
- if `.agents` exists and is empty, proceed to add skills

Because a non-empty `.agents` skips setup entirely, nothing may create that
directory earlier in the launch sequence. In particular the `.claude` →
`.agents` symlink (see [13-gitignore.md](13-gitignore.md)) is created **after**
this step; creating it first would permanently suppress skills installation.
Once linked, `.claude/skills/` resolves to the same skills, so Claude Code and
Pi share one set.

# Adding the skills

1. Determine the skills repo: read `OPS_SKILLS` from `.env.production` if available and not empty, otherwise from `.env`, otherwise fall back to the global `OpsSkills` default (`trustable-ai/skills`)
2. `cd` into the `.agents` directory
2a. Check the source. The per-app `.env` files are writable by the agent, so a changed `OPS_SKILLS` would otherwise make Trustant clone arbitrary repositories whose content becomes agent instructions. The value must match `owner/name`. It is accepted without confirmation only when it equals the global `OpsSkills` default or is listed in the server-side `OPS_SKILLS_ALLOWED` environment variable (comma-separated `owner/name`; app `.env` files cannot extend it). Any other source needs explicit confirmation: the launch-time setup skips it (logged), and `POST /api/skills/<name>/update` refuses with `409` until the request carries `{"confirm_source": "<owner/name>"}` matching the configured source. The check runs before `.agents/skills` is removed.
3. `git clone <repo> skills` — try SSH first (`git@github.com:<repo>.git` with `GIT_SSH_COMMAND` using the ed25519 key and `StrictHostKeyChecking=accept-new` with a dedicated known-hosts file, see `gitSSHCommand`). If SSH fails, clean up and fall back to HTTPS (`https://github.com/<repo>.git`). This is the same pattern used by repo cloning.
4. Remove `skills/.git`
5. `git add .agents/skills`
6. `git commit -m "added skills"`
7. `git push origin`
8. The launch progress bar shows "Setting up agent skills..." while this runs (no separate popup); the launch API still returns `"skills_added": true`

# API

## GET /api/skills/\<name\>

Returns the list of skills and the source repo.

1. Validate `name` with the standard name pattern
2. Check workbench exists
3. Read each subfolder under `.agents/skills/`, parse `SKILL.md` (front matter + markdown body)
4. Return `{"skills": [...], "source": "<repo>"}` where each skill has `name`, `front_matter` (key-value map), and `body` (markdown string)

## POST /api/skills/\<name\>/update

Removes existing skills and re-clones. Optional JSON body `{"confirm_source": "<owner/name>"}` approves a source that is not allow-listed; otherwise such a source returns `409` `{"error": "...", "source": "<owner/name>", "requires_confirmation": true}` and nothing is removed. An invalid `OPS_SKILLS` value returns `400`. The UI shows a confirmation naming the source and retries with `confirm_source` on accept.

1. Check the source (see 2a above)
2. Remove `.agents/skills`
3. Re-run the "adding the skills" process
4. Return `{"status": "updated"}`

# Skills UI

In `app.html` toolbar, show a "Skills" button. On click, show a modal with:

- **Source line**: link to the GitHub repo and an "Update Skills to Latest" button
- **Message**: "Configure OPS_SKILLS in main window to use your own skill repo."
- **Tabbed interface**: one tab per folder under `.agents/skills/`
- Each tab shows:
  - Front matter key-value pairs in a table
  - Body as rendered markdown in a scrollable div

Clicking "Update Skills to Latest" shows a confirmation: "Warning: I will remove current skills and replace with the latest." On accept, calls POST `/api/skills/<name>/update` and reloads the skills list.

# Memory API (AGENTS.md editor)

The toolbar "Memory" modal edits the application's `AGENTS.md`. The backend is
`memory.go`; the route is `/api/memory/<name>`, where `<name>` is the
application name (`^[a-zA-Z][a-zA-Z0-9]{5,19}$`). It requires the normal Trustant
session (see [authentication.md](authentication.md)). The file is
`$WORKBENCH_DIR/<name>/AGENTS.md`.

| Method | Request | Success | Errors |
| --- | --- | --- | --- |
| `GET` | none | `200` `{"content": "<text>"}`; `content` is `""` when the file does not exist | `400` invalid name, `404` workbench not found, `500` read failure |
| `POST` | JSON `{"content": "<text>"}` | `200` `{"status": "ok"}` | `400` invalid name or body, `404` workbench not found, `500` write or `git add` failure |
| other | | | `405` |

`POST` overwrites the whole file (mode `0644`), is idempotent for identical
content, and runs `git add AGENTS.md` in the checkout only when it created the
file. It does not commit. The body has no size limit beyond the server's.

```bash
curl -s -b cookies.txt https://trustant.example.test/api/memory/demoapp1
curl -s -b cookies.txt -X POST -H 'Content-Type: application/json' \
  -d '{"content":"# Notes\n"}' https://trustant.example.test/api/memory/demoapp1
```
