# GenAI metrics must follow the GenAI semantic conventions

`internal/connections/bundledconfig/shared/application/genai_metrics.yaml.tmpl` turns GenAI spans into metrics when `application.genai_metrics` is enabled. The metric names, types, units and attributes come from the OpenTelemetry GenAI semantic conventions, so these metrics can be read the same way as the ones GenAI SDKs emit.

## When this rule applies

Check this rule when a pull request changes any of:

- `internal/connections/bundledconfig/shared/application/genai_metrics.yaml.tmpl`
- `GenAIMetricsConfig` in `internal/config/configschema.go`
- the `signaltometricsconnector` version in `builder-config.yaml`

Only check the metrics and attributes the pull request adds or changes. Don't flag the ones it leaves untouched.

## Where the conventions are

The `main` branch of [open-telemetry/semantic-conventions-genai](https://github.com/open-telemetry/semantic-conventions-genai). Metrics are in `model/gen-ai/metrics.yaml` and `model/gen-ai/token-metrics.yaml`, attributes in `model/gen-ai/registry.yaml`, and changes not yet released in `changelog.d/`.

## What to check

For each metric or attribute the pull request adds or changes:

1. **It matches the conventions.** The metric name, instrument (histogram or counter), unit, attributes, and the operations it covers match `main`.
2. **It is not outdated.** If `main` renamed, replaced or removed it, say so and name what `main` uses instead.
3. **Newer conventions are mentioned.** If `main` has a newer way to record the same data, such as a new metric or attribute, mention it.
4. **The span attributes it reads exist.** Every span attribute used in `value:` or `conditions:` is defined in `main`.
5. **Required attributes are present.** Attributes the conventions mark as required on the metric are on it.
6. **Differences are explained.** A difference from the conventions is fine when the pull request description explains why. Flag only differences with no explanation.
7. **Snapshots are regenerated.** The config snapshot outputs under `internal/commands/config/test/` reflect the change.
