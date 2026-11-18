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

"""Track changes to vendored third-party sources (see AGENTS.md, legal/VENDORED.md).

For each vendored directory this compares every tracked file with the pinned
upstream revision and writes legal/VENDORED.md:

  unmodified  same content as upstream, ignoring this project's license header
  modified    exists upstream with different content
  added       does not exist upstream (written for this project)

and lists upstream files that are no longer present ("removed").

  python legal/vendored_changes.py           # rewrite legal/VENDORED.md
  python legal/vendored_changes.py --check   # exit 1 when it is out of date

Upstream clones are cached in legal/.cache/vendored (Git-ignored, network on
first use).
"""

import hashlib
import os
import re
import subprocess
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
CACHE = os.path.join(ROOT, "legal", ".cache", "vendored")
OUTPUT = os.path.join(ROOT, "legal", "VENDORED.md")

# directory, upstream name, clone URL, pinned revision, license, copyright
VENDORED = [
    ("acp", "obsidian-agent-client", "https://github.com/RAIT-09/obsidian-agent-client.git",
     "0.11.0", "Apache-2.0", "Copyright 2025-2026 RAIT-09"),
    ("acp/pi-acp", "pi-acp", "https://github.com/svkozak/pi-acp.git",
     "9e857dcc05a057404eb1537e5f31e5aef88a5863", "MIT", "Copyright (c) 2025 Sergii Kozak"),
]


def git(*args, cwd=ROOT, binary=False):
    result = subprocess.run(["git", *args], cwd=cwd, capture_output=True)
    if result.returncode:
        sys.exit("git %s failed: %s" % (" ".join(args), result.stderr.decode("utf-8", "replace").strip()))
    return result.stdout if binary else result.stdout.decode("utf-8", "replace")


def upstream_clone(name, url, rev):
    path = os.path.join(CACHE, name)
    if not os.path.isdir(os.path.join(path, ".git")):
        os.makedirs(CACHE, exist_ok=True)
        git("clone", "--quiet", url, path)
    probe = subprocess.run(["git", "cat-file", "-e", rev + "^{commit}"], cwd=path, capture_output=True)
    if probe.returncode:
        git("fetch", "--quiet", "--tags", "origin", cwd=path)
    return path


COMMENT_LINE = re.compile(r"^\s*(#|//)")


def strip_header(data):
    """Remove this project's license header so only real content is compared."""
    text = data.decode("utf-8", "replace").replace("\r\n", "\n")
    prefix = ""
    if text.startswith("#!"):
        first, _, text = text.partition("\n")
        prefix = first + "\n"
    if text.startswith("/*"):
        end = text.find("*/")
        if end != -1 and "Nuvolaris" in text[:end]:
            text = text[end + 2:].lstrip("\n")
    elif COMMENT_LINE.match(text):
        lines = text.split("\n")
        i = 0
        while i < len(lines) and COMMENT_LINE.match(lines[i]):
            i += 1
        if "Nuvolaris" in "\n".join(lines[:i]):
            text = "\n".join(lines[i:]).lstrip("\n")
    return (prefix + text).encode("utf-8")


def digest(data):
    return hashlib.sha256(data.replace(b"\r\n", b"\n")).hexdigest()


def tracked(directory, exclude):
    out = git("ls-files", "-z", "--", directory).split("\0")
    prefix = directory.rstrip("/") + "/"
    files = []
    for path in out:
        if not path.startswith(prefix):
            continue
        rel = path[len(prefix):]
        if any(rel == e or rel.startswith(e + "/") for e in exclude):
            continue
        files.append(rel)
    return sorted(files)


def compare(directory, name, url, rev, exclude):
    clone = upstream_clone(name, url, rev)
    upstream = {}
    for line in git("ls-tree", "-r", rev, cwd=clone).splitlines():
        meta, _, path = line.partition("\t")
        upstream[path] = meta.split()[2]
    result = {"unmodified": [], "modified": [], "added": [], "removed": []}
    local = tracked(directory, exclude)
    for rel in local:
        with open(os.path.join(ROOT, directory, rel), "rb") as handle:
            content = handle.read()
        if rel not in upstream:
            result["added"].append(rel)
            continue
        original = git("cat-file", "blob", upstream[rel], cwd=clone, binary=True)
        same = digest(strip_header(content)) == digest(strip_header(original))
        result["unmodified" if same else "modified"].append(rel)
    result["removed"] = sorted(p for p in upstream if p not in set(local))
    return result


def render():
    lines = [
        "# Vendored sources: change tracking",
        "",
        "Generated by `python legal/vendored_changes.py`; do not edit by hand. It compares",
        "every tracked file of each vendored directory with the pinned upstream revision.",
        "Run it whenever files under `acp/` change and commit the result. See",
        "[AGENTS.md](../AGENTS.md) for the rules and [CREDITS.md](../CREDITS.md) for credits.",
        "",
        "Status meaning: **unmodified** has the same content as upstream (this project's",
        "license header ignored); **modified** differs from upstream; **added** does not",
        "exist upstream; **removed** upstream files that are no longer present.",
        "",
    ]
    for directory, name, url, rev, license_id, copyright_line in VENDORED:
        exclude = ["pi-acp"] if directory == "acp" else []
        result = compare(directory, name, url, rev, exclude)
        lines += [
            "## `%s/`: %s" % (directory, name),
            "",
            "- Upstream: %s" % url[:-4] if url.endswith(".git") else "- Upstream: " + url,
            "- Baseline revision: `%s`" % rev,
            "- License: %s, %s" % (license_id, copyright_line),
            "- Files: %d unmodified, %d modified, %d added, %d removed upstream files"
            % tuple(len(result[k]) for k in ("unmodified", "modified", "added", "removed")),
            "",
        ]
        for key, title in (("modified", "Modified"), ("added", "Added"), ("removed", "Removed")):
            if result[key]:
                lines += ["### %s (%d)" % (title, len(result[key])), ""]
                lines += ["- `%s`" % p for p in result[key]]
                lines.append("")
        lines += ["### Unmodified (%d)" % len(result["unmodified"]), "",
                  "Listed as a count only; every tracked file not named above is identical to upstream.", ""]
    return "\n".join(lines).rstrip("\n") + "\n"


def main():
    text = render()
    if "--check" in sys.argv:
        current = open(OUTPUT, encoding="utf-8", newline="").read().replace("\r\n", "\n") if os.path.exists(OUTPUT) else ""
        if current != text:
            sys.exit("legal/VENDORED.md is out of date: run python legal/vendored_changes.py")
        print("legal/VENDORED.md is up to date")
        return
    with open(OUTPUT, "w", encoding="utf-8", newline="\n") as handle:
        handle.write(text)
    print("wrote legal/VENDORED.md")


if __name__ == "__main__":
    main()
