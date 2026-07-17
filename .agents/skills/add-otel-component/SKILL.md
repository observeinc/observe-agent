---
name: add-otel-component
description: >-
  Add one or more OpenTelemetry Collector components (receivers, processors,
  exporters, connectors, extensions) to the observe-agent distribution.
  Use for ALL requests that mention adding, including, or enabling a new OTel
  component. Triggers: add receiver, add processor, add exporter, add
  connector, add extension, new OTel component, include component, enable
  component, add otel, otelcol component.
---

# Add OTel Component

## Overview

The observe-agent is an OTel Collector distribution built by the
[OpenTelemetry Collector Builder (OCB)](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder).
`builder-config.yaml` in the repo root is the **source of truth** for which
components are included. `observecol/components.go` is auto-generated from it
and must never be edited by hand.

## Step 1: Identify the component

Ask the user (or infer from context):
- The **type**: `receiver`, `processor`, `exporter`, `connector`, or `extension`
- The **Go module path** — for contrib components this follows the pattern
  `github.com/open-telemetry/opentelemetry-collector-contrib/<type>/<name><type>`
  (e.g. `github.com/open-telemetry/opentelemetry-collector-contrib/connector/failoverconnector`,
  `github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver`)

If the user provides a GitHub URL such as
`https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/connector/failoverconnector`,
derive the Go module path directly from the URL path:
`github.com/open-telemetry/opentelemetry-collector-contrib/<type>/<name><type>`.

## Step 2: Determine the correct version

Read `builder-config.yaml` and note the version used by existing contrib
components — look at any `gomod` entry under the same type section, e.g.:

```yaml
connectors:
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector v0.151.0
```

All contrib components use the **same version** as the `dist.version` field.
Use that version for the new component.

## Step 3: Edit builder-config.yaml

Add a new `gomod` line in the correct section (`exporters`, `processors`,
`receivers`, `extensions`, or `connectors`). Keep contrib components grouped
together and sorted alphabetically within the group.

Example — adding `failoverconnector` to the connectors section:

```yaml
connectors:
  - gomod: go.opentelemetry.io/collector/connector/forwardconnector v0.151.0

  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector v0.151.0
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/connector/failoverconnector v0.151.0   # <-- added
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/connector/routingconnector v0.151.0
  - gomod: github.com/open-telemetry/opentelemetry-collector-contrib/connector/spanmetricsconnector v0.151.0
```

## Step 4: Regenerate components.go

Run:

```bash
make build-ocb
```

This runs OCB with `--skip-compilation`, copies
`ocb-build/components.go` → `observecol/components.go`, and runs
`go mod tidy` + `go work vendor` to update all dependency files.

Expected output ends with:
```
INFO    builder/main.go:114     Generating source codes only, the distribution will not be compiled.
```
followed by `go mod tidy` output and `README.md generated successfully.`

**If `make build-ocb` fails because `ocb` is not installed:**

```bash
make install-ocb
```

then retry `make build-ocb`.

## Step 5: Verify

Confirm the new component appears in `observecol/components.go` with three
distinct entries:
1. An import alias at the top of the file
2. A `NewFactory()` call inside `MakeFactoryMap`
3. A build-info map entry with the module path and version

```bash
grep -n "<component-name>" observecol/components.go
```

Expected output (example for `failoverconnector`):
```
16:     failoverconnector "github.com/.../connector/failoverconnector"
262:            failoverconnector.NewFactory(),
272:            failoverconnector.NewFactory().Type(): "github.com/.../connector/failoverconnector v0.151.0",
```

## Step 6: Build and validate

### 6a: Build the agent binary

```bash
make build
```

This compiles the full agent binary including the new component. Expected output ends with:
```
go build -o observe-agent .
```

### 6b: Write a minimal test config

Write a temporary `test-components.yaml` that exercises each new component via a minimal
pipeline. Use `nop` receiver or exporter on the other end of the pipeline as needed.

Rules for the config:
- `token` must be non-empty and contain a colon (e.g. `test:token`)
- `observe_url` must be a valid URL (e.g. `https://123456789.collect.observe-eng.com/`)
- Set `forwarding.enabled`, `self_monitoring.enabled`, and `host_monitoring.enabled` all
  to `false` to avoid unrelated pipeline dependencies during the test
- Add pipelines under `otel_config_overrides.service.pipelines`
- Add the new components under `otel_config_overrides.receivers` / `otel_config_overrides.exporters`
- Use the component's type name (found in its `metadata.yaml` `type:` field), not its Go package name

Example for a new receiver `collectd` (type `collectd`) and exporter `kafka` (type `kafka`):

```yaml
token: "test:token"
observe_url: "https://123456789.collect.observe-eng.com/"

forwarding:
  enabled: false
self_monitoring:
  enabled: false
host_monitoring:
  enabled: false

otel_config_overrides:
  receivers:
    collectd:
      endpoint: 0.0.0.0:8081
    nop:
  exporters:
    nop:
    kafka:
      brokers:
        - localhost:9092
  service:
    pipelines:
      metrics/collectd_test:
        receivers: [collectd]
        exporters: [nop]
      logs/kafka_test:
        receivers: [nop]
        exporters: [kafka]
```

### 6c: Run config validate

```bash
./observe-agent --observe-config ./test-components.yaml config validate
```

Expected output:
```
✅ configuration is valid
```

If validation fails, fix the test config (or the component config) before proceeding.

After a successful validation, remove the temporary config file:
```bash
rm test-components.yaml
```

## Step 7: Propose commit

`make build-ocb` modifies several files. Stage them all:

- `builder-config.yaml` (the source-of-truth edit)
- `observecol/components.go`, `observecol/go.mod`, `observecol/go.sum`
- `go.mod`, `go.sum`
- `vendor/` (updated by `go work vendor`)

Propose a commit message, e.g.:

```
feat: add otel failoverconnector component
```

**⚠️ STOPPING POINT**: Do not `git push` or open a PR without explicit user
confirmation — those are `[MUTATING]` operations.

## Stopping Points

- ✋ **Step 7** — before pushing or opening a PR

## Output

- `builder-config.yaml` updated with new `gomod` entry
- `observecol/components.go` regenerated with new import, factory registration,
  and build-info entry
- `go.mod`, `go.sum`, `observecol/go.mod`, `observecol/go.sum`, `vendor/`
  updated to include the new dependency
