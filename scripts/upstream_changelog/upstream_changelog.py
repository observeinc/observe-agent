#!/usr/bin/env python3
"""Write the OpenTelemetry Collector section of an observe-agent release's notes.

Compares the upstream collector versions in observecol/go.mod at the previous release and at the
release ref, fetches the upstream core and contrib CHANGELOG.md, and lists the entries from every
upstream release in between that touch a module compiled into observecol.

Usage:
    upstream_changelog.py [--from REF] [--to REF]     print the section as Markdown
    upstream_changelog.py --to TAG --update-release   also write it into the GitHub release for TAG
"""
import argparse
import os
import re
import subprocess
import sys
import urllib.error
import urllib.request

from otel_changelog import in_scope, norm, parse_entries, scope_from_gomod

REPOS = {
    "core": ("opentelemetry-collector", "Core"),
    "contrib": ("opentelemetry-collector-contrib", "Contrib"),
}

CATEGORIES = {
    "Breaking changes": "🛑 Breaking changes 🛑",
    "Deprecations": "🚩 Deprecations 🚩",
    "New components": "🚀 New components 🚀",
    "Enhancements": "💡 Enhancements 💡",
    "Bug fixes": "🧰 Bug fixes 🧰",
}

MARK_START = "<!-- upstream-changelog:start -->"
MARK_END = "<!-- upstream-changelog:end -->"

# GitHub rejects release bodies over 125,000 characters; leave room for GoReleaser's changelog.
GITHUB_BODY_LIMIT = 125_000
MAX_CHARS = 100_000


def vkey(version):
    return tuple(int(p) for p in version.split("."))


def collector_versions(gomod_text):
    # Stable modules are versioned v1.x; releases and changelog sections follow the v0.x line.
    core = re.search(r"go\.opentelemetry\.io/collector/otelcol v(0\.\d+\.\d+)\s", gomod_text)
    contrib = re.findall(r"opentelemetry-collector-contrib/\S+ v(0\.\d+\.\d+)\s", gomod_text)
    if not core or not contrib:
        sys.exit("error: observecol/go.mod requires no released go.opentelemetry.io/collector/otelcol "
                 "or opentelemetry-collector-contrib module")
    return {"core": core.group(1), "contrib": max(contrib, key=vkey)}


def split_sections(changelog_text):
    """Map each release in a CHANGELOG.md to its lines. Headings are `## v0.151.0` or `## v1.57.0/v0.151.0`."""
    sections, cur = {}, None
    for line in changelog_text.splitlines():
        if line.startswith("## "):
            versions = re.findall(r"v(0\.\d+\.\d+)\b", line)
            cur = versions[-1] if versions else None
            if cur:
                sections[cur] = []
            continue
        if cur:
            sections[cur].append(line)
    return sections


def fetch_changelog(repo, version):
    url = f"https://raw.githubusercontent.com/open-telemetry/{REPOS[repo][0]}/v{version}/CHANGELOG.md"
    try:
        with urllib.request.urlopen(url, timeout=30) as resp:
            return resp.read().decode()
    except urllib.error.URLError as err:
        sys.exit(f"error: fetching {url}: {err}")


def collect(old, new, gomod_text, fetch):
    contrib_scope, core_scope = scope_from_gomod(gomod_text)
    releases, entries, total = {}, [], 0
    for repo in REPOS:
        releases[repo] = []
        if vkey(new[repo]) <= vkey(old[repo]):
            continue
        sections = split_sections(fetch(repo, new[repo]))
        if new[repo] not in sections:
            sys.exit(f"error: the {repo} CHANGELOG.md at v{new[repo]} has no v{new[repo]} section")
        between = sorted((v for v in sections if vkey(old[repo]) < vkey(v) <= vkey(new[repo])), key=vkey)
        for ver in between:
            found, _ = parse_entries(repo, ver, sections[ver])
            found = [e for e in found if e["block"] not in ("Unmaintained", "Repeat")]
            total += len(found)
            # cmd/* (for example cmd/builder) changes how we build observecol, not the agent customers run.
            entries += [e for e in found if in_scope(repo, e["tags"], contrib_scope, core_scope)
                        and not all(norm(t).startswith("cmd/") for t in e["tags"])]
            releases[repo].append(ver)
    return {"releases": releases, "entries": entries, "total": total}


def link_refs(text, repo):
    url = f"https://github.com/open-telemetry/{REPOS[repo][0]}/issues"

    def linkify(group):
        return re.sub(r"#(\d+)", lambda m: f"[#{m.group(1)}]({url}/{m.group(1)})", group.group(0))

    return re.sub(r"\(#\d+(?:,\s*#\d+)*\)", linkify, text)


def render_entry(e, details):
    lines = [f"- ({REPOS[e['repo']][1]} v{e['ver']}) `{', '.join(e['tags'])}`: {link_refs(e['title'], e['repo'])}"]
    if details:
        lines += [f"  {line}".rstrip() for line in link_refs(e["body"], e["repo"]).strip("\n").splitlines()]
    return "\n".join(lines)


def group_entries(entries):
    order = {repo: i for i, repo in enumerate(REPOS)}
    groups = {heading: [] for heading in CATEGORIES}
    for e in sorted(entries, key=lambda e: (order[e["repo"]], vkey(e["ver"]))):
        groups.setdefault(e["heading"], []).append(e)
    return [(heading, items) for heading, items in groups.items() if items]


def describe(versions):
    if versions["core"] == versions["contrib"]:
        return f"v{versions['core']}"
    return f"core v{versions['core']} and contrib v{versions['contrib']}"


def render_sections(old, new, changes, groups, all_details, hidden):
    """Render the section. Without all_details, only breaking changes keep their detail lines."""
    out = [
        "## OpenTelemetry Collector changes",
        "",
        f"This release updates the OpenTelemetry Collector from {describe(old)} to {describe(new)}. "
        "It includes changes from these upstream releases:",
        "",
    ]
    for repo, versions in changes["releases"].items():
        if versions:
            base = f"https://github.com/open-telemetry/{REPOS[repo][0]}/releases/tag"
            out.append(f"- {REPOS[repo][1]}: " + ", ".join(f"[v{v}]({base}/v{v})" for v in versions))
    out += [
        "",
        "The lists below cover only the components and shared packages built into the Observe Agent: "
        f"{len(changes['entries'])} of {changes['total']} upstream entries.",
    ]
    if not all_details:
        out += ["", "To stay under GitHub's release size limit, this section left out the details of non-breaking "
                    "entries. The linked upstream release notes have the full text."]
    for heading, items in groups:
        count = len(items)
        out += ["", f"### {CATEGORIES.get(heading, heading)}", "", "<details>",
                f"<summary>{count} change{'' if count == 1 else 's'}</summary>", ""]
        shown = items[: count - hidden.get(heading, 0)]
        details = all_details or heading == "Breaking changes"
        out += [render_entry(e, details) for e in shown]
        if len(shown) < count:
            out.append(f"- …and {count - len(shown)} more in the upstream release notes.")
        out += ["", "</details>"]
    return "\n".join(out) + "\n"


def render(old, new, changes, from_ref, budget=MAX_CHARS):
    if not any(changes["releases"].values()):
        return ("## OpenTelemetry Collector changes\n\n"
                f"This release is still based on OpenTelemetry Collector {describe(new)}, the same as {from_ref}.\n")
    groups = group_entries(changes["entries"])
    out = render_sections(old, new, changes, groups, True, {})
    if len(out) <= budget:
        return out
    hidden = {}
    out = render_sections(old, new, changes, groups, False, hidden)
    droppable = [(heading, len(items)) for heading, items in reversed(groups) if heading != "Breaking changes"]
    for heading, count in droppable:
        while len(out) > budget and hidden.get(heading, 0) < count:
            hidden[heading] = hidden.get(heading, 0) + 1
            out = render_sections(old, new, changes, groups, False, hidden)
    return out


def splice(body, section):
    block = f"{MARK_START}\n{section.strip()}\n{MARK_END}"
    if MARK_START in body and MARK_END in body:
        return body[: body.index(MARK_START)] + block + body[body.index(MARK_END) + len(MARK_END):]
    return body.rstrip("\n") + "\n\n" + block + "\n"


def run(*args, stdin=None):
    return subprocess.run(args, check=True, capture_output=True, text=True, input=stdin).stdout


def previous_tag(ref):
    return run("git", "describe", "--tags", "--abbrev=0", "--match", "v[0-9]*", "--exclude", "*-*", f"{ref}^").strip()


def update_release(tag, render_for_budget):
    body = run("gh", "release", "view", tag, "--json", "body", "-q", ".body")
    rest = splice(body, "")
    section = render_for_budget(min(MAX_CHARS, GITHUB_BODY_LIMIT - len(rest)))
    run("gh", "release", "edit", tag, "--notes-file", "-", stdin=splice(body, section))
    return section


def main():
    parser = argparse.ArgumentParser(description=__doc__.split("\n\n")[0])
    parser.add_argument("--to", default="HEAD", help="release ref (default: HEAD)")
    parser.add_argument("--from", dest="from_ref", help="previous release ref (default: the latest v* tag before --to)")
    parser.add_argument("--update-release", action="store_true",
                        help="write the section into the GitHub release named by --to, replacing any earlier copy")
    args = parser.parse_args()

    os.chdir(run("git", "rev-parse", "--show-toplevel").strip())
    from_ref = args.from_ref or previous_tag(args.to)
    new_mod = run("git", "show", f"{args.to}:observecol/go.mod")
    old = collector_versions(run("git", "show", f"{from_ref}:observecol/go.mod"))
    new = collector_versions(new_mod)
    changes = collect(old, new, new_mod, fetch_changelog)
    print(f"{from_ref} -> {args.to}: collector {describe(old)} -> {describe(new)}; "
          f"{len(changes['entries'])} of {changes['total']} upstream entries in scope", file=sys.stderr)

    if args.update_release:
        print(update_release(args.to, lambda budget: render(old, new, changes, from_ref, budget)), end="")
    else:
        print(render(old, new, changes, from_ref), end="")


if __name__ == "__main__":
    main()
