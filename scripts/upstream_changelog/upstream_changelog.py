#!/usr/bin/env python3
"""Write the OpenTelemetry Collector section of an observe-agent release's notes.

Compares the upstream collector versions in observecol/go.mod at the previous release and at the
release ref, fetches the upstream core and contrib CHANGELOG.md, and lists the entries from every
upstream release in between that touch a module compiled into observecol.

Usage:
    upstream_changelog.py [--from REF] [--to REF]     print the section as Markdown
    upstream_changelog.py --to TAG --update-release   also write it into the GitHub release for TAG
    upstream_changelog.py --from TAG --to TAG --agent-releases --max-chars N
        start with the notes of the observe-agent releases in between (used by the agent Helm chart)
    upstream_changelog.py --list-releases CURRENT     versions on main newer than CURRENT (no gh)
    upstream_changelog.py --fetch-notes CURRENT TARGET DIR
        write DIR/{core,contrib}-<ver>.md for (CURRENT, TARGET] from CHANGELOG.md and CHANGELOG-API.md
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
    "Known Issues": "❗ Known Issues ❗",
}

AGENT_REPO_URL = "https://github.com/observeinc/observe-agent"
RAW_ROOT = "https://raw.githubusercontent.com/open-telemetry"
NOTES_NAME = re.compile(r"^(core|contrib)-.+\.md$")

MARK_START = "<!-- upstream-changelog:start -->"
MARK_END = "<!-- upstream-changelog:end -->"

# GitHub rejects release bodies over 125,000 characters; leave room for GoReleaser's changelog.
GITHUB_BODY_LIMIT = 125_000
MAX_CHARS = 100_000


def vkey(version):
    return tuple(int(p) for p in version.split("."))


def warn(message):
    prefix = "::warning::" if os.environ.get("GITHUB_ACTIONS") == "true" else "warning: "
    print(prefix + message, file=sys.stderr)


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


def fetch_file(repo, ref, filename, missing_ok=False):
    url = f"{RAW_ROOT}/{REPOS[repo][0]}/{ref}/{filename}"
    try:
        with urllib.request.urlopen(url, timeout=30) as resp:
            return resp.read().decode()
    except urllib.error.HTTPError as err:
        if missing_ok and err.code == 404:
            return ""
        sys.exit(f"error: fetching {url}: {err}")
    except urllib.error.URLError as err:
        sys.exit(f"error: fetching {url}: {err}")


def fetch_changelog(repo, version):
    return fetch_file(repo, f"v{version}", "CHANGELOG.md")


def versions_between(sections, current, target=None):
    lo, hi = vkey(current), vkey(target) if target else (10**9,)
    return sorted((v for v in sections if lo < vkey(v) <= hi), key=vkey)


def rebuild_notes(user, api):
    """Rebuild one `gh release view` body from the End User and API changelog slices."""
    body = "## End User Changelog\n\n" + "\n".join(user).strip("\n") + "\n"
    api_text = "\n".join(api).strip("\n") if api else ""
    if api_text:
        body += "\n## API Changelog\n\n" + api_text + "\n"
    return body


def list_releases(current, fetch=None):
    """Print core and contrib 0.x versions newer than current, newest first."""
    if fetch is None:
        fetch = fetch_file
    lines = []
    for repo in REPOS:
        newer = versions_between(split_sections(fetch(repo, "main", "CHANGELOG.md")), current)
        lines.append(f"{repo}: {' '.join(reversed(newer)) or 'up to date'}")
    return "\n".join(lines) + "\n"


def fetch_notes(current, target, out, fetch=None):
    """Write parse_changelog.py's per-version files for every release in (current, target]."""
    if fetch is None:
        fetch = fetch_file
    os.makedirs(out, exist_ok=True)
    for name in os.listdir(out):
        if NOTES_NAME.fullmatch(name):
            os.remove(os.path.join(out, name))
    written = []
    for repo in REPOS:
        user = split_sections(fetch(repo, "main", "CHANGELOG.md"))
        api = split_sections(fetch(repo, "main", "CHANGELOG-API.md", missing_ok=True))
        for ver in versions_between(user, current, target):
            if not "".join(user[ver]).strip():
                sys.exit(f"error: {repo} v{ver} has an empty CHANGELOG.md section")
            path = os.path.join(out, f"{repo}-{ver}.md")
            with open(path, "w") as f:
                f.write(rebuild_notes(user[ver], api.get(ver, [])))
            written.append(path)
    return written


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
            found, unparsed = parse_entries(repo, ver, sections[ver])
            for bullet in unparsed:
                warn(f"{repo} v{ver}: skipped a bullet that isn't a changelog entry: {bullet['text']}")
            if not found:
                warn(f"{repo} v{ver}: found no changelog entries; check whether the CHANGELOG.md format changed")
            found = [e for e in found if e["block"] not in ("Unmaintained", "Repeat")]
            total += len(found)
            # cmd/* (for example cmd/builder) changes how we build observecol, not the agent customers run.
            entries += [e for e in found if in_scope(repo, e["tags"], contrib_scope, core_scope)
                        and not all(norm(t).startswith("cmd/") for t in e["tags"])]
            releases[repo].append(ver)
    return {"releases": releases, "entries": entries, "total": total}


def link_issue_refs(text, issues_url):
    def linkify(group):
        return re.sub(r"#(\d+)", lambda m: f"[#{m.group(1)}]({issues_url}/{m.group(1)})", group.group(0))

    return re.sub(r"\(#\d+(?:,\s*#\d+)*\)", linkify, text)


def link_refs(text, repo):
    return link_issue_refs(text, f"https://github.com/open-telemetry/{REPOS[repo][0]}/issues")


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


def agent_tags_between(tags, from_ref, to_ref):
    """Release tags after from_ref up to to_ref, newest first. An untagged to_ref takes every later release."""
    def version(tag):
        m = re.fullmatch(r"v(\d+\.\d+\.\d+)", tag)
        return vkey(m.group(1)) if m else None

    low, high = version(from_ref), version(to_ref)
    if low is None:
        sys.exit(f"error: --agent-releases needs --from to name a vX.Y.Z release tag, not {from_ref}")
    found = [t for t in tags if version(t) and version(t) > low and (high is None or version(t) <= high)]
    return sorted(found, key=version, reverse=True)


def clean_agent_notes(body):
    """Strip the collector section and make references link to observe-agent from any repository's release."""
    if MARK_START in body and MARK_END in body:
        body = body[: body.index(MARK_START)] + body[body.index(MARK_END) + len(MARK_END):]
    lines = []
    for line in body.strip().splitlines():
        if line.strip() == "## Changelog":
            continue
        m = re.match(r"^(#+)(\s.*)$", line)
        if m:
            # Stay below the "### vX.Y.Z" heading that render_agent_releases adds for each release.
            lines.append("#" * max(len(m.group(1)) + 1, 4) + m.group(2))
            continue
        line = re.sub(r"^(\s*[*-]\s+)([0-9a-f]{7,40})(?=:)",
                      lambda c: f"{c.group(1)}[{c.group(2)[:7]}]({AGENT_REPO_URL}/commit/{c.group(2)})", line)
        lines.append(link_issue_refs(line, f"{AGENT_REPO_URL}/issues"))
    return "\n".join(lines).strip()


def render_agent_releases(from_ref, to_ref, releases, budget=MAX_CHARS):
    """List each release's notes, newest first, leaving out the oldest releases when over budget."""
    head = ["## Observe Agent changes", "", f"This release updates the Observe Agent from {from_ref} to {to_ref}."]
    blocks = [["", f"### [{tag}]({AGENT_REPO_URL}/releases/tag/{tag})", "", clean_agent_notes(body) or "No release notes."]
              for tag, body in releases]
    for kept in range(len(blocks), -1, -1):
        out = head + [line for block in blocks[:kept] for line in block]
        left_out = [tag for tag, _ in releases[kept:]]
        if left_out:
            links = ", ".join(f"[{tag}]({AGENT_REPO_URL}/releases/tag/{tag})" for tag in left_out)
            out += ["", f"To stay under GitHub's release size limit, this section left out the notes of "
                        f"{len(left_out)} release{'' if len(left_out) == 1 else 's'}: {links}"]
        text = "\n".join(out) + "\n"
        if len(text) <= budget:
            break
    return text


def render_notes(old, new, changes, from_ref, to_ref, agent_releases, budget):
    """Render the optional Observe Agent section and the collector section, sharing one character budget.

    When both are long, the collector section is trimmed to half the budget first and the agent notes get
    the rest; the collector section then grows back into whatever the agent notes leave unused.
    """
    agent_md = ""
    if agent_releases is not None:
        collector_share = len(render(old, new, changes, from_ref, budget // 2))
        agent_md = render_agent_releases(from_ref, to_ref, agent_releases, budget - collector_share - 1) + "\n"
    out = agent_md + render(old, new, changes, from_ref, budget - len(agent_md))
    if len(out) > budget:
        sys.exit(f"error: the release notes need {len(out)} characters even after trimming, "
                 f"more than the {budget} allowed")
    return out


def run(*args, stdin=None):
    try:
        return subprocess.run(args, check=True, capture_output=True, text=True, input=stdin).stdout
    except FileNotFoundError:
        sys.exit(f"error: {args[0]} not found")
    except subprocess.CalledProcessError as err:
        sys.exit(f"error: {' '.join(args)} failed with exit status {err.returncode}:\n{err.stderr.strip()}")


def previous_tag(ref):
    return run("git", "describe", "--tags", "--abbrev=0", "--match", "v[0-9]*", "--exclude", "*-*", f"{ref}^").strip()


def require_tag(ref):
    if not run("git", "tag", "--list", ref).strip():
        sys.exit(f"error: --update-release needs --to to name a release tag, not {ref}")


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
    parser.add_argument("--agent-releases", action="store_true",
                        help="start with the notes of every observe-agent release after --from up to --to")
    parser.add_argument("--max-chars", type=int, default=MAX_CHARS,
                        help=f"longest output to produce (default: {MAX_CHARS})")
    parser.add_argument("--list-releases", metavar="CURRENT",
                        help="list core and contrib versions newer than CURRENT from CHANGELOG.md on main")
    parser.add_argument("--fetch-notes", nargs=3, metavar=("CURRENT", "TARGET", "NOTES_DIR"),
                        help="write NOTES_DIR/{core,contrib}-<ver>.md for versions in (CURRENT, TARGET]")
    args = parser.parse_args()

    if args.list_releases:
        print(list_releases(args.list_releases), end="")
        return
    if args.fetch_notes:
        for path in fetch_notes(*args.fetch_notes):
            print(path)
        return

    os.chdir(run("git", "rev-parse", "--show-toplevel").strip())
    if args.update_release:
        require_tag(args.to)
    from_ref = args.from_ref or previous_tag(args.to)
    agent_releases = None
    if args.agent_releases:
        tags = agent_tags_between(run("git", "tag", "--list", "v*").split(), from_ref, args.to)
        agent_releases = [(t, run("gh", "release", "view", t, "--json", "body", "-q", ".body")) for t in tags]
    new_mod = run("git", "show", f"{args.to}:observecol/go.mod")
    old = collector_versions(run("git", "show", f"{from_ref}:observecol/go.mod"))
    new = collector_versions(new_mod)
    changes = collect(old, new, new_mod, fetch_changelog)
    print(f"{from_ref} -> {args.to}: collector {describe(old)} -> {describe(new)}; "
          f"{len(changes['entries'])} of {changes['total']} upstream entries in scope", file=sys.stderr)

    def build(budget):
        return render_notes(old, new, changes, from_ref, args.to, agent_releases, min(budget, args.max_chars))

    if args.update_release:
        print(update_release(args.to, build), end="")
    else:
        print(build(args.max_chars), end="")


if __name__ == "__main__":
    main()
