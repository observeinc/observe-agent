---
name: check-otel-updates
description: Use when checking for newer OpenTelemetry Collector (otelcol core or contrib) releases, planning or reviewing an OTel upgrade of the observe-agent, or judging how upstream changelog entries affect the agent, the helm chart, or the public docs.
---

# Check OTel Updates (devbot)

Written for the Baby Bots `devbot`. Follow `base/SKILL.md` with the differences
below; its impact checks, verification levels, and report contract are
unchanged. Read `base/SKILL.md` for the steps, but run every script from the
observe-agent checkout: the installed `base/` copy has no `scripts/` folder
for `parse_changelog.py` to import from.

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

1. **Versions.** From the observe-agent checkout, run
   `python3 scripts/upstream_changelog/upstream_changelog.py --list-releases <current>`
   instead of `gh release list`.
2. **Notes.** Run
   `python3 scripts/upstream_changelog/upstream_changelog.py --fetch-notes <current> <target> /tmp/otel-notes`.
   It rebuilds each release body from the same `CHANGELOG.md` / `CHANGELOG-API.md`
   fetch the release workflow uses, and writes a file only for versions that
   repo has. It fails when neither repo has a `<target>` section. The rebuilt
   body lacks the "Unmaintained Components" list, which the parser drops
   anyway.
3. **Ledger.** From the observe-agent checkout, run
   `python3 .agents/skills/check-otel-updates/parse_changelog.py /tmp/otel-notes
   observecol/go.mod /tmp/otel-audit`. If a file yields no
   entries because its bullets lack the `:` after the component (contrib
   `v0.165.1` does), classify those bullets by hand, list them in the
   appendix, delete the file, and rerun.
4. **Build plumbing.** Instead of `gh pr view`, find the PR number in the
   notes for the patch's repo
   (`grep -nE '#<n>\b' /tmp/otel-notes/<core|contrib>-*.md`), or read the
   patched file at the target tag on `raw.githubusercontent.com`. If neither
   settles it, report the item as not verified, with the steps to check it.
