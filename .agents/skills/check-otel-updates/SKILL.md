---
name: check-otel-updates
description: >-
  Check for updates to OpenTelemetry Collector components and dependencies.
  Use when the user asks about upgrading OTel, checking for new collector
  versions, or reviewing upstream OpenTelemetry changes.
---

# Check OTel Updates

## Overview

This skill checks whether newer versions of the OpenTelemetry Collector are
available compared to what this project currently uses, then performs a **full
examination** of every changelog entry between the two versions and classifies
each change by its real impact on our build, our emitted telemetry, and our
configuration.

**Core principle: OpenTelemetry's changelog headings are unreliable. Never
assume a change is safe because it is not under a "Breaking changes" heading.**
Behavior-breaking changes are routinely filed under `Enhancements` and `Bug
fixes`, components get removed or renamed, and metric names/units/attributes
change silently. Read and classify *every* entry that touches a component we
ship — by its effect, not by the heading it sits under.

## Why full examination is required

The release notes (`gh release view` body is identical to each repo's
`CHANGELOG.md`) have several traps that defeat a heading-only reading:

- **Breaking changes hide under other headings.** A rate-limiter rewrite, a
  latency-threshold semantics change, a UCUM metric-unit fix, or new config
  validation that fails startup will often appear under `💡 Enhancements` or
  `🧰 Bug fixes`, not `🛑 Breaking changes`.
- **Two changelog blocks per release.** Each release body has an **End User
  Changelog** and an **API Changelog**, each with its *own* Breaking /
  Deprecation / Enhancement / Bug-fix headings. Scan both.
- **Silent telemetry changes.** Renamed or removed metrics, changed units,
  added/renamed/removed attributes, and changed values break dashboards and
  downstream content with no error and no "breaking" tag.
- **Components get removed or renamed.** A component we ship can be deleted
  entirely (its module 404s at the new tag) or renamed with a new module path.
- **Components are renamed to snake_case.** Many labels changed (e.g.
  `metricstransform` → `metrics_transform`). Search both spellings.

Classify every change by impact, not by the heading it appears under.

## Step 1: Determine the current version

Read `builder-config.yaml` in the repo root. The current OTel version is in
the `dist.version` field (e.g. `version: 0.146.0`). All core and contrib
components should use this same version.

## Step 2: Fetch latest releases

Use `gh` (GitHub CLI) to check for releases newer than the current version in
**both** repositories:

```bash
# Core collector
gh release list --repo open-telemetry/opentelemetry-collector --limit 10

# Contrib collector
gh release list --repo open-telemetry/opentelemetry-collector-contrib --limit 10
```

Compare the release tags against the current version. If no newer versions
exist, report that the project is up to date and stop.

## Step 3: Fetch and fully examine the release notes

For every version between the current version (exclusive) and the latest, fetch
the **complete** release body for both repos and save it so you can grep it:

```bash
mkdir -p /tmp/otel-notes
for v in <each intermediate version> <latest>; do
  gh release view "v$v" --repo open-telemetry/opentelemetry-collector \
    --json body -q .body > "/tmp/otel-notes/core-$v.md"
  gh release view "v$v" --repo open-telemetry/opentelemetry-collector-contrib \
    --json body -q .body > "/tmp/otel-notes/contrib-$v.md"
done
```

Do **not** read only the `🛑 Breaking changes` sections. Extract **every**
bullet — across `Breaking changes`, `Deprecations`, `Enhancements`, and `Bug
fixes`, in **both** the End User and API changelog blocks — that names a
component we ship or is project-wide (`all`, `pkg/*`, `cmd/*`), then read and
classify each one by impact (see Step 5).

Extract per component like this, matching both the current spelling and the
snake_case variant of each leaf name from `builder-config.yaml`:

```bash
# <tokens> = alternation of every component leaf name (both spellings),
# e.g. tail_sampling|tailsampling|metrics_transform|metricstransform|...
grep -nE '^- `(processor|receiver|exporter|connector|extension)/(<tokens>)`|^- `(all|pkg/[a-z]+|cmd/[a-z]+)`' /tmp/otel-notes/*.md
```

Cross-check the count against the authoritative `CHANGELOG.md` at the target tag
if anything looks thin — the `gh` body should match it entry-for-entry.

## Step 4: Filter to components we use

Read `builder-config.yaml` in the repo root to get the full list of components.
Each section (`exporters`, `processors`, `receivers`, `extensions`,
`connectors`) contains `gomod` entries whose module path identifies the
component. Extract the component name from each module path (e.g.
`github.com/open-telemetry/opentelemetry-collector-contrib/exporter/fileexporter`
→ `exporter/file`).

Only report changes that affect these components **or** are project-wide /
general changes (tagged `all`, `pkg/confmap`, `pkg/otelcol`, `pkg/service`,
`cmd/builder`, etc.). Skip our own custom components under
`github.com/observeinc/observe-agent/`.

### Verify every shipped component still exists at the target version

A component we ship may have been **removed** or **renamed** upstream — this is
a hard blocker that no changelog heading reliably announces. For each contrib
component in `builder-config.yaml`, confirm its module still resolves at the
target tag. A `404` means it was removed and must be dropped from
`builder-config.yaml` and `go.mod` (or replaced):

```bash
curl -s -o /dev/null -w '%{http_code}\n' \
  "https://raw.githubusercontent.com/open-telemetry/opentelemetry-collector-contrib/v<target>/<component-path>/go.mod"
```

## Step 5: Produce a summary

Classify every extracted change by its **impact**, regardless of which changelog
heading it came from. Produce these sections:

### Blockers

Changes that stop the upgrade or the build: components we ship that were
**removed or renamed** (from Step 4), and any **minimum Go version** bump.

### Breaking changes (by effect, not by heading)

Anything that breaks our build, changes config semantics, changes a default, or
can **fail startup** (e.g. stricter validation, removed/renamed config fields).
Pull these from *all* sections — Enhancements and Bug fixes included. Treat
policy/algorithm rewrites (e.g. a sampler switching to a token bucket, a metric
changing how it is computed) as behavior changes even when they are listed as
enhancements. For each:
1. What changed
2. Whether it requires action in our config or code
3. Suggested remediation if applicable

### Silent telemetry changes

Changes to **emitted data** that break dashboards/content with no error and no
"breaking" tag: metric renames/removals, **unit changes** (e.g. UCUM fixes),
added/renamed/removed attributes, and changed values. These are the easiest to
miss and most often live under Enhancements/Bug fixes.

### Deprecations

Deprecations relevant to our components with migration guidance (note the many
snake_case component renames; old aliases usually still work for now).

### Notable enhancements

Briefly list enhancements to our components that may be useful.

### Bug fixes

Bug fixes for our components, especially ones that change behavior or fix issues
we may have encountered.

## Step 6: Check every module's go.mod (not just `components/`)

This is a multi-module repo. Besides the root `go.mod` and `observecol/go.mod`,
custom and patched modules under `components/` **and `patches/`** carry their
own OTel dependency versions — the `patches/` modules are easy to forget. Find
them all (exclude vendored deps):

```bash
find . -name go.mod -not -path './vendor/*'
```

For each one, compare the OTel library versions (modules under
`go.opentelemetry.io/collector/` and
`github.com/open-telemetry/opentelemetry-collector-contrib/`) against the
versions in the root `go.mod`. Flag any that have diverged.

If the target upgrade version differs from the root `go.mod`, note that the
root, `observecol`, and **all** `components/*` and `patches/*` `go.mod` files
must be updated together.

## Step 7: Recommend next steps

Based on the analysis:
- Call out any **blockers** first (removed/renamed components, Go version bump) —
  these decide whether the upgrade is even possible in one step.
- If there are breaking changes, flag the upgrade as requiring careful review
  and list specific files/configs that may need updates.
- Surface **silent telemetry changes** separately: they need dashboard/content
  updates on the backend, not just agent config changes.
- If the upgrade is straightforward, say so.
- Always recommend running the full test suite after upgrading.
