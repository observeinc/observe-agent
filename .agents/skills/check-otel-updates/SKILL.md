---
name: check-otel-updates
description: Use when checking for newer OpenTelemetry Collector (otelcol core or contrib) releases, planning or reviewing an OTel upgrade of the observe-agent, or judging how upstream changelog entries affect the agent, the helm chart, or the public docs.
---

# Check OTel Updates

## Overview

Find newer collector releases, then decide what every changelog entry between our version and the target does to **what we publish**: the agent's bundled configs and Go code, the helm chart, the public docs, and the feature-gate flags we document.

**Core principle: judge each entry by its effect on our published surfaces, verify that effect in upstream source or by running it, and give every in-scope entry a written verdict.** Changelog headings are unreliable. Behavior-breaking changes regularly appear under *Enhancements*, *Bug fixes*, and *Deprecations* (a component deletion was once filed as a deprecation), and feature-gate promotions silently flip defaults.

## Inputs: the published surfaces

Audit the **released** refs, not local feature branches. Snapshot each with `git archive <ref> | tar -x -C /tmp/otel-audit/<name>` after `git fetch`.

| Surface | Repo and ref | What to read |
|---|---|---|
| Agent bundled configs | `observe-agent` `origin/main` | `internal/connections/bundledconfig/**` (includes `shared/common/internal_telemetry.yaml.tmpl`) |
| Agent Go code | same | `internal/commands/status/statusretriever.go` (hard-coded self-metric names), `observecol/feature_gates.go`, custom components |
| Build plumbing | same | `builder-config.yaml`, every `go.mod`, `go.work` replaces, `patches/*` |
| Helm chart | `helm-charts` `origin/main` | `charts/agent/templates/**`, `charts/agent/values.yaml` (feature-gate `extraArgs`) |
| Public docs | `observe-docs-repo` default branch (`origin/v1.0`); also `observe-docs-current` `origin/v1.0` | Every YAML block with `receivers:`/`processors:`/`exporters:`/`connectors:` |

If a repo isn't checked out, say in the report that the surface was not audited. Don't treat it as having no impact.

## Steps

1. **Versions.** The current version is `dist.version` in `builder-config.yaml`. List releases with `gh release list --repo open-telemetry/opentelemetry-collector{,-contrib} --limit 15`. If nothing is newer, report "up to date" and stop.
2. **Notes.** Save every intermediate body to `/tmp/otel-notes/{core,contrib}-<ver>.md` using `gh release view v<ver> --repo … --json body -q .body`.
3. **Ledger.** Run `python3 .agents/skills/check-otel-updates/parse_changelog.py /tmp/otel-notes observecol/go.mod /tmp/otel-audit`. Scope comes from `observecol/go.mod`, so the shared `pkg/*` and `internal/*` packages count, not only `builder-config.yaml` components. Read every line of `compact.txt`, and read full bodies in `ledger.json` for anything that isn't obviously opt-in. Open `unparsed.txt`: every line must be a non-entry (for example, an Unmaintained component list). If a real entry appears there, fix the parser before classifying.
4. **Removed modules.** For each contrib module in `builder-config.yaml`, `curl -s -o /dev/null -w '%{http_code}' https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib/v<target>/<path>/go.mod`. A 404 is a blocker; bisect versions to find the release that removed it.
5. **Impact checks.** Run every check in the table below against every surface. Each check is required.
6. **Classify.** Give every ledger entry exactly one category, with a one-line reason that names the surface or evidence. A "No impact" verdict also needs a reason.
7. **Report.** Write it in the shape described under "Report contract". The appendix lists every entry.

## Required impact checks

| Trap | How to check |
|---|---|
| **Feature-gate lifecycle** | For every gate an entry mentions, read its stage at our version (vendored `.../internal/metadata/generated_feature_gates.go`) and at the target (component `metadata.yaml` at the tag). alpha→beta means the default flips on: treat it as a behavior change. beta→stable means it can no longer be disabled. Removed means any flag naming it fails startup (`ApplyFeatureGates` returns the error). Grep all surfaces for `feature-gates`/`featureGates`. |
| **"Already on by default at our version"** | This is *not* "no impact". The upgrade removes the opt-out. Grep docs and configs for anything that depends on the old behavior (for example, `invert_match` relying on inverted tail-sampling decisions). |
| **Core `pkg/service` / telemetry entries** | We set `service::telemetry::metrics::readers` explicitly. Any change to defaults "when explicitly configured" hits us, even if it's filed as a bug fix. Run the current binary twice on a local port (shipped settings, then the new defaults set explicitly) and diff `/metrics` names against `statusretriever.go`. |
| **OTTL semantics** | Grep every OTTL statement on all surfaces. Look for `set(` without a `where … != nil` guard (nil handling), converters like `Int(`/`Double(` (parse errors), typed datapoint paths, and maps fed to `ToKeyValueString`/`XXH128` (identity hashes). Check which `transform`/`filter`/`routing` instances lack `error_mode`, since that default changed. |
| **Default output of components we configure** | For each metric, attribute, or unit an entry changes, grep the surfaces for it (for example `system.cpu.utilization`, `cpu.usage`, `otelcol_`). Resource detectors in use (`detectors:`) gain or lose attributes and API calls. |
| **Removed or stricter config** | Grep the surfaces for each removed key or newly validated setting. A hit in docs counts: customers copy docs. |
| **Rollout and rollback** | Hash-ring or routing changes (`loadbalancing`) split traces while old and new versions run side by side. Storage or checkpoint format changes need forward *and* backward read compatibility, so read both versions' loader code. |
| **Build plumbing** | `go.work` replaces are pinned to the old version and silently stop applying after a bump. For each `patches/*` module, check whether the upstream fix shipped (`gh pr view`) and whether the patched file still exists at that path at the target tag. Compare the `go` directive in every module against the new minimum Go version. |
| **Go API** | Grep our Go code (excluding `vendor/` and generated `components.go`) for each symbol the API changelog removes or renames. |
| **Conditional rendering** | Before calling an impact "default", find the `{{ if … }}` that renders the processor and the value's default in `values.yaml` or the agent config schema. Report which path is affected: default install or opt-in flag (for example `externalWriteHashes`, which turns on the collector-side `observe_identifiers_hash` and `observe_labels_hash`). For any hash built from attributes, list every upgrade change that adds, removes, or re-values an attribute it covers. |
| **Test coverage** | For each break, check whether a test would actually fail. Tests that serve static fixtures (for example `internal/commands/status/testfixtures/testmetrics`) keep passing after a rename. Say which check exposes the break, or that none does. |

## Verification levels

Every reported item states one of these:

- **Ran it:** reproduced with our binary or a small Go harness against the target module.
- **Upstream source:** read the code at the tag on `raw.githubusercontent.com`.
- **Changelog:** taken from the release notes only.

If you can't verify an item, say so and include the exact reproduction steps. Never infer semantics from an entry's title (for example, whether an attribute lands on the resource or the datapoint).

## Report contract

Write the report as a file (`otel-upgrade-report-v<cur>-to-v<target>.md`) with these sections, in order:

1. **Verdict:** the few changes that decide the upgrade, in one short list.
2. **Method:** surfaces with repo, ref, and commit SHA; entry counts; and a count per category.
3. **Blockers:** removed or renamed modules, plumbing that silently breaks, and the Go minimum version.
4. **Breaks or changes something we ship.** For each: what changed, where (`file:line` on each surface), the effect, the fix, and the verification level.
5. **Silent telemetry changes**, split into self-monitoring, labels/values, resource attributes, and pipeline behavior.
6. **Feature-gate table:** gate, status at the target, whether a flag now fails startup, and which gates newly default on.
7. **Customer-authored config only:** the release-note material.
8. **Deprecations** we use.
9. **Behavior-changing fixes** on our default paths.
10. **Plan per repo:** agent, helm, docs, content, and tests.
11. **Appendix:** every ledger entry with ID, version, upstream heading, component, category, and reason.

Categories: Blocker · Breaks or changes something we ship · Silent telemetry change · Feature-gate lifecycle · Customer-authored config only · Deprecation · Behavior-changing fix on a default path · No impact.

## Rationalizations seen in practice

| Excuse | Reality |
|---|---|
| "It isn't under Breaking changes" | Deletions, unit changes, and default flips ship under every heading. |
| "The gate is already on by default at our version" | The upgrade removes the opt-out, and our docs or configs may still depend on the old behavior. |
| "It's a bug fix" | A core telemetry "fix" renamed every self-monitoring metric for explicitly configured readers. |
| "Our transform probably handles it" | Read the upstream code; don't guess where data lands. |
| "Only the agent templates matter" | The helm chart and the docs ship collector config too, and so do the documented `--feature-gates` flags. |
| "Too many entries to read" | `parse_changelog.py` makes the ledger; every entry still gets a verdict. |

**Red flags:** you're about to mark something "no impact" without naming a surface you grepped; you skipped a gate entry because it says "stabilize" or "promote"; you haven't opened helm or docs; or the appendix has fewer rows than `ledger.json`.

## Notes

- `parse_changelog.py` derives scope from the *current* `observecol/go.mod`. Modules that only enter the dependency tree after the bump (for example a new `config/*` module) show up through the entries of the components that adopt them.
- Recommend the full test suite, `/pre-commit-smoke-test` (including `observe-agent status`), and regenerated config snapshots after any upgrade.
