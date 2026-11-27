#!/usr/bin/env python3
# Copyright 2025-2026 Nuvolaris Inc
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published
# by the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program.  If not, see <https://www.gnu.org/licenses/>.

"""Compare this fork with the original trustant/trustant and its gitlinks.

Writes UPSTREAM-COMPARISON.md in the repository root:

  1. the gitlinks of the original repository (remote `upstream`) against this
     fork, with the baseline commit that was vendored and how far the original
     gitlink has moved;
  2. for each former gitlink, every file changed since the vendored baseline,
     with the reason (hand-written in REASONS below, otherwise the commit
     subjects that touched the file) and the full line diff when it is short;
  3. a per-file summary of the root repository against `upstream/main`.

  python legal/upstream_compare.py             # fetch upstream, rewrite the report
  python legal/upstream_compare.py --offline   # no network: skip fetch and gitlink distance
  python legal/upstream_compare.py --check     # exit 1 when the report is out of date

Run it whenever vendored sources change, after an upstream sync, and with
legal/vendored_changes.py. Add a REASONS entry for every new modified file.
"""

import os
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
OUTPUT = os.path.join(ROOT, "UPSTREAM-COMPARISON.md")
CACHE = os.path.join(ROOT, "legal", ".cache", "gitlinks")
UPSTREAM = "upstream/main"
UPSTREAM_URL = "https://github.com/trustant/trustant"
MAX_DIFF_LINES = 120          # longer diffs get a pointer only
LOCKFILES = ("package-lock.json", "bun.lock")

# path, remote, baseline (the gitlink commit the fork vendored), commit before vendoring, note
COMPONENTS = [
    ("acp", "https://github.com/trustant/trustant-acp",
     "9786f796380aa9b2000d8cebd913fa41150c89ed", "b06dc8d^1", "vendored 2026-11-19 (b06dc8d)"),
    ("acp/pi-acp", "https://github.com/trustant/pi-acp",
     "4e526e5f64eaec647d78c919ba9dc071197c2517", "b06dc8d^1", "nested in acp, vendored 2026-11-19 (b06dc8d)"),
    ("oplugins", "https://github.com/trustable-ai/openserverless-task",
     "27b281eb7615c9094f1c6edab10a2b4fe2fd621c", "c41c63c^", "vendored 2026-11-24 (c41c63c)"),
    ("skills", "https://github.com/trustable-ai/skills",
     "56d349b6174e53707723ed839c252995122dc46d", "c41c63c^", "vendored 2026-11-24 (c41c63c)"),
    ("mcp", "https://github.com/trustant/openserverless-mcp",
     "baf14007f9106cfd63b1396fd5d374f7dd9b7537", "c41c63c^", "vendored 2026-11-24 (c41c63c)"),
]
LOCAL_TAG = {"oplugins": "vendor/oplugins-baseline", "skills": "vendor/skills-baseline",
             "mcp": "vendor/mcp-baseline"}

# Why each modified file differs from the baseline. Key: "<component>/<file>".
REASONS = {
    "acp/esbuild.web.mjs": "Embeds the shared Trustant favicon in the standalone browser bundle.",
    "acp/web/index.html": "Declares the embedded Trustant favicon for direct browser access.",
    "acp/SPEC.md": "Documents the shared favicon in the embedded browser bundle.",
    "oplugins/admin/docopts.md": "Documents the OPS_ADMIN_PASSWORD environment variable added to `admin adduser` (see opsfile.yml): the password no longer has to appear in the process list. Plus the Apache 4(b) modification notice.",
    "oplugins/admin/opsfile.yml": "Security: `adduser` reads the password from OPS_ADMIN_PASSWORD when set (keeps it out of argv and the process list), falls back to the positional argument, then unsets the variable (commit a0bd211, #119). User deletion tolerates an already absent user. Includes the Apache 4(b) modification notice.",
    "oplugins/admin/ingress/opsfile.yml": "Ingress deletion tolerates resources already removed so a retry can finish cleaning up the app. Includes the Apache 4(b) modification notice.",
    "oplugins/debug/runtime/invoke.py": "Code quality: `!= None` becomes `is not None`; bare `except:` becomes `except Exception:` (a bare except also swallows KeyboardInterrupt and SystemExit); a comment clarifies payload handling.",
    "oplugins/ide/deploy/package.json": "Dependency update: glob 14.0.2 (commit 64a17b2).",
    "oplugins/ide/deploy/bun.lock": "Lockfile regenerated for the glob 14.0.2 update (commit 64a17b2).",
    "oplugins/prereq.yml": "Supply chain: the new `download-verified` task pins and sha256-verifies the Linux Kubernetes tool downloads instead of fetching them unchecked. Fork note added in the header.",
    "oplugins/setup/kubernetes/DEBUG.txt": "Only the Apache 4(b) modification notice in the first line; the whole file shows as changed because its line endings differ from the baseline (CRLF/LF).",
    "oplugins/util/upload/.env.test": "Removed: a test environment file that should not be vendored (not needed by the Trustant build).",
    "mcp/package.json": "Dependency update: zod 4.6.7 (commit d254cb9, Dependabot alerts).",
    "mcp/package-lock.json": "Lockfile regenerated for the zod 4.6.7 update (commit d254cb9).",
    "mcp/requirements.txt": "Python pins updated for Dependabot alerts (certifi 2026.10.18, dnspython 2.9.2 and others; commit d254cb9).",
    "mcp/tests/invoke.test.ts": "The test writes a POSIX shebang stub for `ops`, which Windows cannot execute, so it is skipped on win32 (commit 11f03a7, #125 #126). Plus the Apache 4(b) modification notice.",
    "acp/pi-acp/package.json": "Dependency bump for the SDK and Pi major upgrade (commit d767604, 'Major fase 3'); repository, bugs and homepage point to trustant/pi-acp, the fork's origin.",
    "acp/pi-acp/package-lock.json": "Lockfile regenerated for the same upgrade.",
    "acp/pi-acp/src/acp/sdk-compat.ts": "New file: compatibility layer for the ACP SDK 2.x API used by the fork.",
    "acp/pi-acp/src/acp/agent.ts": "Adapted to the SDK compatibility layer.",
    "acp/pi-acp/src/acp/session.ts": "Adapted to the SDK compatibility layer.",
    "acp/pi-acp/src/index.ts": "Entry point adapted to the SDK compatibility layer.",
    "acp/pi-acp/src/pi-rpc/launch-config.ts": "Change to the `_meta.trustant.piLaunch` launch contract (see acp/pi-acp/TRUSTANT_FORK.md).",
    "acp/pi-acp/test/helpers/fakes.ts": "Test fake updated for the SDK change.",
    "acp/pi-acp/test/unit/launch-config.test.ts": "Test for the launch-config change.",
    "acp/pi-acp/.github/workflows/ci.yml": "Removed: workflow of the standalone repository; the root repository's CI covers it.",
    "acp/pi-acp/.github/workflows/github-release.yml": "Removed: release workflow of the standalone repository (unused inside this repository).",
    "acp/pi-acp/.github/workflows/npm-publish.yml": "Removed: npm publishing is not used; Trustant packages the source directly.",
}


def git(*args, cwd=ROOT, check=True):
    r = subprocess.run(["git", *args], cwd=cwd, capture_output=True)
    if r.returncode and check:
        sys.exit("git %s failed: %s" % (" ".join(args), r.stderr.decode("utf-8", "replace").strip()))
    return r.stdout.decode("utf-8", "replace") if not r.returncode else ""


def short(sha):
    return sha[:7]


def upstream_gitlinks():
    links = {}
    for line in git("ls-tree", "-r", UPSTREAM).splitlines():
        meta, path = line.split("\t", 1)
        mode, _kind, sha = meta.split()
        if mode == "160000":
            links[path] = sha
    return links


def distance(name, remote, ours, theirs, offline):
    """(ahead, behind) of the fork baseline against the original gitlink, or None."""
    if offline or not theirs:
        return None
    repo = os.path.join(CACHE, name.replace("/", "_"))
    if not os.path.isdir(repo):
        os.makedirs(CACHE, exist_ok=True)
        subprocess.run(["git", "clone", "-q", "--bare", "--filter=blob:none", remote + ".git", repo],
                       capture_output=True)
    if not os.path.isdir(repo):
        return None
    subprocess.run(["git", "fetch", "-q", "origin", "+refs/heads/*:refs/heads/*"], cwd=repo, capture_output=True)
    out = git("rev-list", "--left-right", "--count", "%s...%s" % (ours, theirs), cwd=repo, check=False).split()
    return (int(out[0]), int(out[1])) if len(out) == 2 else None


def describe(d, offline):
    if d is None:
        return "n/a (offline)" if offline else "unknown"
    ahead, behind = d
    if (ahead, behind) == (0, 0):
        return "same commit"
    return "baseline is %d commit(s) behind and %d commit(s) ahead of it" % (behind, ahead)


def tree_of(path, sha):
    return LOCAL_TAG.get(path, sha) + "^{tree}"


def subjects(rev_range, path, limit=4):
    out = git("log", "--no-merges", "--format=%h %ad %s", "--date=short", rev_range, "--", path, check=False)
    lines = out.strip().splitlines()
    more = len(lines) - limit
    lines = lines[:limit]
    if more > 0:
        lines.append("... and %d more" % more)
    return lines


def component_section(c, upstream_links, offline):
    path, remote, baseline, since, note = c
    theirs = upstream_links.get(path)
    out = ["## `%s/`\n" % path,
           "- Origin: %s" % remote,
           "- Baseline vendored by the fork: `%s` (%s)" % (baseline, note)]
    if theirs:
        out.append("- Gitlink in the original repository: `%s`; %s" % (theirs, describe(distance(path, remote, baseline, theirs, offline), offline)))
    else:
        out.append("- Gitlink in the original repository: none (nested inside the `acp` gitlink)")
    tree = tree_of(path, baseline)
    status = {}
    for line in git("diff", "--name-status", tree, "HEAD:" + path).splitlines():
        st, f = line.split("\t", 1)
        status[f] = {"M": "modified", "A": "added", "D": "removed"}.get(st[0], st)
    rows = []
    for line in git("diff", "--numstat", tree, "HEAD:" + path).splitlines():
        a, d, f = line.split("\t", 2)
        if path == "acp" and f.startswith("pi-acp/"):
            continue
        rows.append((a, d, f))
    rows.sort(key=lambda r: r[2])
    out.append("- Files changed since the baseline: %d; every other file is identical to the baseline.\n" % len(rows))
    if not rows:
        out.append("No file differs from the baseline: the directory is identical to the original gitlink commit.\n")
        return "\n".join(out)
    out.append("| File | Status | +/- | Why |\n|---|---|---|---|")
    for a, d, f in rows:
        key = "%s/%s" % (path, f)
        why = REASONS.get(key)
        if not why:
            subj = subjects(since + "..HEAD", key)
            if subj:
                why = "(auto) " + "; ".join(s.replace("|", "/") for s in subj)
            else:
                why = "(auto) changed by the vendoring commit itself (%s): standalone-repository file dropped or replaced by the root equivalent" % note
        out.append("| `%s` | %s | +%s/-%s | %s |" % (f, status.get(f, "?"), a, d, why))
    out.append("")
    if path == "acp":
        out.append("The `acp/` delta is too large to reproduce line by line. Per-file diff: "
                   "`git diff %s HEAD:acp -- <file>`.\n" % tree)
        return "\n".join(out)
    out.append("### Line diffs\n")
    for a, d, f in rows:
        if f.endswith(LOCKFILES):
            out.append("`%s`: lockfile, +%s/-%s lines, not reproduced.\n" % (f, a, d))
            continue
        extra = ["--ignore-cr-at-eol"] if f.endswith("DEBUG.txt") else []
        diff = git("diff", "--no-color", "-U1", *extra, tree, "HEAD:" + path, "--", f, check=False)
        body = ["" if line == " " else line for line in diff.splitlines()[4:]]
        if not body:
            out.append("#### `%s`\n\nNo textual change besides line endings or deletion.\n" % f)
        elif len(body) > MAX_DIFF_LINES:
            out.append("`%s`: %d diff lines, not reproduced (`git diff %s HEAD:%s -- %s`).\n" % (f, len(body), tree, path, f))
        else:
            out.append("#### `%s`\n\n```diff\n%s\n```\n" % (f, "\n".join(body)))
    return "\n".join(out)


def root_section():
    # A commit that only refreshes this report must not make it stale again.
    report = ":!UPSTREAM-COMPARISON.md"
    revision = git("log", "-1", "--format=%H", "HEAD", "--", ".", report).strip()
    base = git("merge-base", revision, UPSTREAM).strip()
    ahead, behind = git("rev-list", "--left-right", "--count", revision + "..." + UPSTREAM).split()
    excl = [":!" + c[0] for c in COMPONENTS if "/" not in c[0]] + [":!oplugins-truinst", report]
    numstat = git("diff", "--numstat", base, revision, "--", ".", *excl).splitlines()
    status = {}
    for line in git("diff", "--name-status", base, revision, "--", ".", *excl).splitlines():
        st, f = line.split("\t", 1)
        status[f] = st[0]
    up = set(git("diff", "--name-only", base, UPSTREAM).splitlines())
    out = ["## Root repository against `%s`\n" % UPSTREAM,
           "- Merge base `%s`; the fork has %s commits upstream lacks, upstream has %s the fork lacks." % (short(base), ahead, behind),
           "- Files also changed upstream since the merge base are marked **conflict risk**.\n",
           "| File | St | +/- | Conflict risk | Latest fork commits touching it (the reason) |\n|---|---|---|---|---|"]
    for line in sorted(numstat, key=lambda row: row.split("\t", 2)[2]):
        a, d, f = line.split("\t", 2)
        subj = subjects(base + ".." + revision, f, 2)
        out.append("| `%s` | %s | +%s/-%s | %s | %s |" % (f, status.get(f, "?"), a, d, "**yes**" if f in up else "",
                                                       "<br>".join(s.replace("|", "/") for s in subj)))
    out.append("")
    return "\n".join(out)


def gitlink_table(links, offline):
    out = ["## Gitlinks: original repository against this fork\n",
           "| Path | Original gitlink | Fork now | Vendored baseline | Baseline against the original gitlink |",
           "|---|---|---|---|---|"]
    comp = {c[0]: c for c in COMPONENTS}
    for p in sorted(set(links) | {c[0] for c in COMPONENTS if "/" not in c[0]}):
        theirs = links.get(p)
        if p in comp:
            c = comp[p]
            out.append("| `%s` | `%s` | directory | `%s` | %s |" % (p, short(theirs or "-"), short(c[2]),
                       describe(distance(p, c[1], c[2], theirs, offline), offline)))
        else:
            mine = git("ls-tree", "HEAD", p).split()
            sha = mine[2] if len(mine) >= 3 and mine[1] == "commit" else "-"
            out.append("| `%s` | `%s` | gitlink `%s` | same | %s |" % (p, short(theirs or "-"), short(sha),
                       "identical" if sha == theirs else "differs from the original gitlink"))
    out.append("")
    out.append("`acp/pi-acp` has no gitlink of its own in the original repository: it is nested inside `acp`.\n")
    return "\n".join(out)


def build(offline):
    if not offline:
        subprocess.run(["git", "fetch", "-q", "upstream"], cwd=ROOT, capture_output=True)
    links = upstream_gitlinks()
    parts = ["# Comparison with the original repository and its gitlinks\n",
             "Original: %s (`%s`). Generated by `python legal/upstream_compare.py`; do not edit by hand. "
             "Hand-written reasons live in `REASONS` in that script.\n" % (UPSTREAM_URL, UPSTREAM),
             "Keep this report current: regenerate it whenever vendored directories change (together with "
             "`legal/vendored_changes.py`), after any upstream sync, and before a release. See [AGENTS.md](AGENTS.md), "
             "[legal/VENDORED.md](legal/VENDORED.md) and [acp/UPSTREAM.md](acp/UPSTREAM.md). "
             "`python legal/upstream_compare.py --check` fails when it is out of date.\n",
             "Scope: line diffs are reproduced for every vendored file except lockfiles and diffs longer than %d lines. "
             "`acp/` and the root repository are summarised per file; the reason comes from commit subjects "
             "(an `(auto)` prefix means no hand-written reason exists yet).\n" % MAX_DIFF_LINES,
             gitlink_table(links, offline)]
    for c in COMPONENTS:
        parts.append(component_section(c, links, offline))
    parts.append(root_section())
    return "\n".join(parts)


def main():
    offline = "--offline" in sys.argv
    text = build(offline)
    if "--check" in sys.argv:
        cur = open(OUTPUT, encoding="utf-8").read() if os.path.exists(OUTPUT) else ""
        if cur != text:
            sys.exit("UPSTREAM-COMPARISON.md is out of date: run python legal/upstream_compare.py")
        return
    with open(OUTPUT, "w", encoding="utf-8", newline="\n") as f:
        f.write(text)
    print("wrote", OUTPUT)


if __name__ == "__main__":
    main()
