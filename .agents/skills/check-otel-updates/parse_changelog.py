#!/usr/bin/env python3
"""Parse OTel collector release notes into a ledger of entries that touch our build.

Usage:
    parse_changelog.py NOTES_DIR OBSERVECOL_GO_MOD OUT_DIR

NOTES_DIR holds core-<ver>.md and contrib-<ver>.md (bodies from `gh release view`).
Scope comes from the modules required by observecol/go.mod, so shared pkg/* and
internal/* packages are included, not only the components in builder-config.yaml.

Writes OUT_DIR/ledger.json (every in-scope entry, with full body text),
OUT_DIR/compact.txt (one line per entry) for classification, and
OUT_DIR/unparsed.txt (top-level bullets that didn't match the entry format).
"""
import glob
import json
import os
import sys

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(os.path.abspath(__file__)), "..", "..", ".."))
sys.path.insert(0, os.path.join(REPO_ROOT, "scripts", "upstream_changelog"))

from otel_changelog import in_scope, parse_entries, scope_from_gomod  # noqa: E402


def parse(paths):
    entries, unparsed, per_file = [], [], {}
    for path in paths:
        name = os.path.basename(path)
        repo, ver = name[:-3].split("-", 1)
        with open(path) as f:
            found, bad = parse_entries(repo, ver, f)
        entries += found
        unparsed += [dict(u, file=name) for u in bad]
        per_file[name] = len(found)
    return entries, unparsed, per_file


def main():
    notes_dir, gomod, out = sys.argv[1:4]
    paths = sorted(glob.glob(os.path.join(notes_dir, "*-*.md")))
    if not paths:
        sys.exit(f"error: no release notes (core-<ver>.md / contrib-<ver>.md) found in {notes_dir}")
    with open(gomod) as f:
        contrib, core = scope_from_gomod(f.read())
    entries, unparsed, per_file = parse(paths)
    for name, count in per_file.items():
        print(f"{name}: {count} entries", file=sys.stderr)
    empty = [name for name, count in per_file.items() if count == 0]
    if empty:
        sys.exit(f"error: no changelog entries parsed from {', '.join(empty)} in {notes_dir}; "
                 "re-fetch the release notes (step 2) or check the note format")
    ours = [e for e in entries
            if e["block"] not in ("Unmaintained", "Repeat") and in_scope(e["repo"], e["tags"], contrib, core)]
    seen = set()
    ledger = []
    deduped = 0
    # The API changelog repeats End User entries; keep the End User copy. chloggen writes one
    # entry per component, so the tags are part of the key.
    for e in ours:
        key = (e["repo"], e["ver"], tuple(e["tags"]), tuple(e["prs"]), e["title"][:60])
        if e["block"] == "API" and key in seen:
            deduped += 1
            continue
        seen.add(key)
        e["id"] = f"{'core' if e['repo'] == 'core' else 'ctb'}{e['ver'].split('.')[1]}-{len(ledger):03d}"
        ledger.append(e)
    os.makedirs(out, exist_ok=True)
    json.dump(ledger, open(os.path.join(out, "ledger.json"), "w"), indent=1)
    with open(os.path.join(out, "compact.txt"), "w") as f:
        for e in ledger:
            f.write(f"{e['id']} {e['block']}/{e['heading']} {','.join(e['tags'])}: {e['title'][:170]}\n")
    with open(os.path.join(out, "unparsed.txt"), "w") as f:
        for u in unparsed:
            f.write(f"{u['file']}:{u['line']} [{u['block']}] {u['text']}\n")
    outside_unmaintained = sum(1 for u in unparsed if u["block"] != "Unmaintained")
    print(f"parsed {len(entries)} entries; {len(ledger)} in scope ({deduped} API duplicates dropped) "
          f"-> {out}/ledger.json, {out}/compact.txt")
    print(f"{len(unparsed)} unparsed top-level bullets ({outside_unmaintained} outside Unmaintained sections) -> {out}/unparsed.txt",
          file=sys.stderr)


if __name__ == "__main__":
    main()
