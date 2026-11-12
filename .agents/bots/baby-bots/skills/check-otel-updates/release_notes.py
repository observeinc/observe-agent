#!/usr/bin/env python3
"""List OTel collector releases and rebuild their notes from the changelogs.

Usage:
    release_notes.py list CURRENT
    release_notes.py fetch CURRENT TARGET NOTES_DIR

Reads CHANGELOG.md and CHANGELOG-API.md of core and contrib from
raw.githubusercontent.com (main). A release body is the End User slice plus the
API slice, so `fetch` writes NOTES_DIR/{core,contrib}-<ver>.md in the shape
`parse_changelog.py` reads, for every version in (CURRENT, TARGET] that the repo
has. `list` prints the versions newer than CURRENT per repo, newest first.
"""

import os
import re
import sys
import urllib.request

REPOS = {"core": "opentelemetry-collector", "contrib": "opentelemetry-collector-contrib"}
RAW = "https://raw.githubusercontent.com/open-telemetry/{repo}/main/{file}"
HEADING = re.compile(r"^## (?:v\d+\.\d+\.\d+/)?v(0\.\d+\.\d+)\s*$")


def version_key(ver):
    return tuple(int(p) for p in ver.split("."))


def slices(text):
    """Map each `## v<stable>/v<ver>` section's 0.x version to its body."""
    out, ver = {}, None
    for line in text.splitlines():
        m = HEADING.match(line)
        if m:
            ver = m.group(1)
            out[ver] = []
        elif line.startswith("## "):
            ver = None
        elif ver is not None:
            out[ver].append(line)
    return {v: "\n".join(body).strip("\n") for v, body in out.items()}


def notes(user, api):
    """The release body for one version; an absent API slice is omitted."""
    body = "## End User Changelog\n\n" + user + "\n"
    if api.strip():
        body += "\n## API Changelog\n\n" + api + "\n"
    return body


def between(versions, current, target):
    lo, hi = version_key(current), version_key(target)
    return sorted((v for v in versions if lo < version_key(v) <= hi), key=version_key)


def fetch(repo, file):
    with urllib.request.urlopen(RAW.format(repo=repo, file=file), timeout=60) as resp:
        return resp.read().decode()


def main(argv):
    if len(argv) == 3 and argv[1] == "list":
        for name, repo in REPOS.items():
            user = slices(fetch(repo, "CHANGELOG.md"))
            newer = between(user, argv[2], "0.9999.0")
            print(f"{name}: {' '.join(reversed(newer)) or 'up to date'}")
        return 0
    if len(argv) == 5 and argv[1] == "fetch":
        current, target, out = argv[2:]
        os.makedirs(out, exist_ok=True)
        for name, repo in REPOS.items():
            user = slices(fetch(repo, "CHANGELOG.md"))
            api = slices(fetch(repo, "CHANGELOG-API.md"))
            for ver in between(user, current, target):
                if not user[ver].strip():
                    sys.exit(f"error: {name} v{ver} has an empty CHANGELOG.md section")
                path = os.path.join(out, f"{name}-{ver}.md")
                with open(path, "w") as f:
                    f.write(notes(user[ver], api.get(ver, "")))
                print(path)
        return 0
    sys.exit(__doc__)


if __name__ == "__main__":
    sys.exit(main(sys.argv))
