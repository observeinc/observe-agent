---
name: check-otel-updates
description: Use when checking for newer OpenTelemetry Collector (otelcol core or contrib) releases, planning or reviewing an OTel upgrade of the observe-agent, or judging how upstream changelog entries affect the agent, the helm chart, or the public docs.
---

# Check OTel Updates (Baby Bots)

Written for the Baby Bots agent. Follow `base/SKILL.md` with the differences
below; its impact checks, verification levels, and report contract are
unchanged. Paths are relative to this skill's directory.

## Tools

- There is no `gh` and no `api.github.com`. The network reaches
  `raw.githubusercontent.com`, `github.com` over git, `proxy.golang.org`, and
  `sum.golang.org`.
- Run Go builds and the agent binary through `dev-env exec --` when the
  checkout has a prepared environment.

## Inputs

The session checks out each surface at its released ref: `observe-agent`
(`main`), `helm-charts` (`main`), `observe-docs-repo` (`v1.0`), and
`observe-docs-current` (`v1.0`). Read those checkouts in place; do not
`git fetch` or `git archive`. Record `git -C <checkout> rev-parse HEAD` for
the Method section. A surface with no checkout is "not audited".

## Step changes

1. **Versions.** Run `python3 release_notes.py list <current>` instead of
   `gh release list`.
2. **Notes.** Run
   `python3 release_notes.py fetch <current> <target> /tmp/otel-notes`. It
   rebuilds each release body from the repo's `CHANGELOG.md` and
   `CHANGELOG-API.md`, and writes a file only for versions that repo has. The
   rebuilt body lacks the "Unmaintained Components" list, which the parser
   drops anyway.
3. **Ledger.** Run `python3 base/parse_changelog.py /tmp/otel-notes
   <observe-agent>/observecol/go.mod /tmp/otel-audit`. If a file yields no
   entries because its bullets lack the `:` after the component (contrib
   `v0.165.1` does), classify those bullets by hand, list them in the
   appendix, delete the file, and rerun.
4. **Build plumbing.** Instead of `gh pr view`, find the PR number in the
   notes (`grep -rn '#<n>' /tmp/otel-notes`), or read the patched file at the
   target tag on `raw.githubusercontent.com`. If neither settles it, report
   the item as not verified, with the steps to check it.
