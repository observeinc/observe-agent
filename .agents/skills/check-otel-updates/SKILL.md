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
available compared to what this project currently uses, then analyzes release
notes for breaking changes and relevant new features.

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

## Step 3: Read release notes for each new version

For every version between the current version (exclusive) and the latest, fetch
the release notes:

```bash
gh release view <tag> --repo open-telemetry/opentelemetry-collector
gh release view <tag> --repo open-telemetry/opentelemetry-collector-contrib
```

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

## Step 5: Produce a summary

For each new version, produce a report with the following sections:

### Breaking Changes

List every breaking change that affects our components or is project-wide.
For each one, explain:
1. What changed
2. Whether it requires action in our config or code
3. Suggested remediation if applicable

### Deprecations

List deprecations relevant to our components with migration guidance.

### Notable Enhancements

Briefly list enhancements to our components that may be useful.

### Bug Fixes

List bug fixes for our components, especially any that fix issues we may have
encountered.

### Minimum Go Version

Note if the minimum Go version requirement changed.

## Step 6: Check custom component go.mod files

The `components/` directory contains custom OTel components, each with its own
`go.mod`. Find all `go.mod` files under `components/`:

```bash
find components -name go.mod
```

For each one, compare the OTel library versions (modules under
`go.opentelemetry.io/collector/` and
`github.com/open-telemetry/opentelemetry-collector-contrib/`) against the
versions in the root `go.mod`. Flag any that have diverged.

If the target upgrade version differs from the root `go.mod`, note that both
the root and component `go.mod` files will need to be updated together.

## Step 7: Recommend next steps

Based on the analysis:
- If there are breaking changes, flag the upgrade as requiring careful review
  and list specific files/configs that may need updates.
- If the upgrade is straightforward, say so.
- Always recommend running the full test suite after upgrading.
