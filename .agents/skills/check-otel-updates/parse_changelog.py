#!/usr/bin/env python3
"""Parse OTel collector release notes into a ledger of entries that touch our build.

Usage:
    parse_changelog.py NOTES_DIR OBSERVECOL_GO_MOD OUT_DIR

NOTES_DIR holds core-<ver>.md and contrib-<ver>.md (bodies from `gh release view`).
Scope comes from the modules required by observecol/go.mod, so shared pkg/* and
internal/* packages are included, not only the components in builder-config.yaml.

Writes OUT_DIR/ledger.json (every in-scope entry, with full body text) and
OUT_DIR/compact.txt (one line per entry) for classification.
"""
import glob
import json
import os
import re
import sys

KINDS = ("receiver", "processor", "exporter", "connector", "extension")

# Changelog tags that don't match their module path.
ALIASES = {
    "pkg/coreinternal": "internal/coreinternal",
    "pkg/fileconsumer": "pkg/stanza",
    "pkg/prometheus": "pkg/translator/prometheus",
    "extension/storage/filestorage": "extension/filestorage",
    "extension/storage": "extension/filestorage",
    "processor/transformprocessor/internal/logparsingfuncs": "processor/transform",
}


def norm(tag):
    tag = ALIASES.get(tag.strip(), tag.strip())
    parts = tag.split("/")
    if parts[0] in KINDS and len(parts) > 1:
        leaf = parts[1].replace("_", "")
        for kind in KINDS:
            if leaf.endswith(kind) and leaf != kind:
                leaf = leaf[: -len(kind)]
        parts[1] = leaf
        if parts[0] == "extension" and parts[1] == "storage" and len(parts) > 2:
            parts = ["extension", parts[2]]
    return ALIASES.get("/".join(parts), "/".join(parts))


def scope_from_gomod(path):
    text = open(path).read()
    contrib = {norm(m) for m in re.findall(r"opentelemetry-collector-contrib/([a-z0-9_/]+) v\d", text)}
    core = {norm(m) for m in re.findall(r"go\.opentelemetry\.io/collector/([a-z0-9_/]+) v\d", text)}
    return contrib, core


def in_scope(repo, tags, contrib, core):
    for t in tags:
        n = norm(t)
        if n == "all":
            return True
        if repo == "core":
            if n.startswith(("pkg/", "cmd/builder", "provider/")) or n in core:
                return True
            continue
        if n in contrib or any(n.startswith(m + "/") or m.startswith(n + "/") for m in contrib if m.startswith(("pkg/", "internal/"))):
            return True
    return False


def parse(notes_dir):
    entries = []
    for path in sorted(glob.glob(os.path.join(notes_dir, "*-*.md"))):
        repo, ver = os.path.basename(path)[:-3].split("-", 1)
        block, heading, cur = "End User", "", None
        for line in open(path):
            line = line.rstrip("\n")
            if line.startswith("## "):
                low = line.lower()
                block = ("API" if "api" in low else
                         "End User" if "end" in low and "user" in low else
                         "Unmaintained" if "unmaintained" in low else
                         "Repeat" if re.match(r"## v\d", line) else block)
                cur = None
                continue
            if line.startswith("### "):
                heading = re.sub(r"[^A-Za-z ]", "", line).strip()
                cur = None
                continue
            m = re.match(r"^- `([^`]+)`:\s*(.*)$", line)
            if m:
                cur = dict(repo=repo, ver=ver, block=block, heading=heading,
                           tags=[t.strip() for t in m.group(1).split(",")],
                           title=m.group(2), body="", prs=re.findall(r"#(\d+)", m.group(2)))
                entries.append(cur)
            elif cur is not None and (line.startswith("  ") or not line):
                cur["body"] += line.strip() + "\n"
            else:
                cur = None
    return entries


def main():
    notes_dir, gomod, out = sys.argv[1:4]
    contrib, core = scope_from_gomod(gomod)
    entries = parse(notes_dir)
    ours = [e for e in entries
            if e["block"] not in ("Unmaintained", "Repeat") and in_scope(e["repo"], e["tags"], contrib, core)]
    seen = set()
    ledger = []
    for i, e in enumerate(ours):
        key = (e["repo"], e["ver"], tuple(e["prs"]), e["title"][:60])
        if e["block"] == "API" and key in seen:
            continue
        seen.add(key)
        e["id"] = f"{'core' if e['repo'] == 'core' else 'ctb'}{e['ver'].split('.')[1]}-{i:03d}"
        ledger.append(e)
    os.makedirs(out, exist_ok=True)
    json.dump(ledger, open(os.path.join(out, "ledger.json"), "w"), indent=1)
    with open(os.path.join(out, "compact.txt"), "w") as f:
        for e in ledger:
            f.write(f"{e['id']} {e['block']}/{e['heading']} {','.join(e['tags'])}: {e['title'][:170]}\n")
    print(f"parsed {len(entries)} entries; {len(ledger)} in scope -> {out}/ledger.json, {out}/compact.txt")


if __name__ == "__main__":
    main()
