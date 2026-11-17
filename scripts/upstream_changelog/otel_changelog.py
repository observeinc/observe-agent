"""Parse OpenTelemetry Collector changelog entries and decide which ones touch the observecol build.

Shared by scripts/upstream_changelog/upstream_changelog.py (release notes) and
.agents/skills/check-otel-updates/parse_changelog.py (upgrade audits).

Scope comes from the modules required by observecol/go.mod, so shared pkg/* and internal/*
packages count, not only the components in builder-config.yaml.
"""
import re

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


def scope_from_gomod(text):
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


def parse_entries(repo, ver, lines):
    """Parse one release's notes or CHANGELOG.md section.

    Returns (entries, unparsed). Each entry records the "## " block it sat in (End User, API,
    Unmaintained, or Repeat for a previous release's section) and its "### " heading. Body lines
    keep their indentation relative to the entry so nested lists still render.
    """
    entries, unparsed = [], []
    block, heading, cur = "End User", "", None
    for lineno, line in enumerate(lines, 1):
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
        elif cur is not None and (line.startswith("  ") or not line.strip()):
            cur["body"] += line[2:].rstrip() + "\n"
        else:
            if line.startswith("- "):
                unparsed.append(dict(line=lineno, block=block, text=line))
            cur = None
    return entries, unparsed
