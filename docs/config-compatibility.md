# Config Compatibility: Renaming Bundled OTel Components Without Breaking Users

This document explains why the `otlphttp` -> `otlp_http` style renames were a breaking
change for users, how other OpenTelemetry distributions avoid that class of breakage,
and the mechanism we should adopt so future renames are absorbed automatically.

## The problem

We renamed the component types in our bundled config to follow the upstream
`lower_snake_case` convention:

| Before                              | After                                 |
| ----------------------------------- | ------------------------------------- |
| `otlphttp/observe`                  | `otlp_http/observe`                   |
| `otlphttp/observemetrics`           | `otlp_http/observemetrics`            |
| `otlphttp/observetracing`           | `otlp_http/observetracing`            |
| `otlphttp/agentheartbeat`           | `otlp_http/agentheartbeat`            |
| `hostmetrics/host-monitoring-host`  | `host_metrics/host-monitoring-host`   |
| `hostmetrics/host-monitoring-process` | `host_metrics/host-monitoring-process` |
| `filelog/host_monitoring`           | `file_log/host_monitoring`            |
| `filestats/agent`                   | `file_stats/agent`                    |

It is worth being precise about what actually broke, because the obvious explanation is
wrong. Upstream did **not** remove the old names. Each renamed component keeps its old
name as a deprecated alias (implemented via `componentalias.TypeAliasHolder` on the
component factory), so a config that says `otlphttp:` still resolves to the same exporter
and will keep doing so for a long deprecation window. On its own, the upstream rename
would have been invisible to our users.

What broke is ours. `otel_config_overrides` is merged into the bundled config by the
confmap resolver, and **confmap merges maps by literal key string**. It has no idea that
`otlphttp` and `otlp_http` name the same component type. So when we changed the bundled
key and a user's override still says the old one:

```yaml
otel_config_overrides:
  exporters:
    otlphttp/observe:
      sending_queue:
        num_consumers: 20
```

the resolver does not merge the override into our exporter. It creates a **second,
unrelated component** called `otlphttp/observe` whose entire config is the two lines the
user wrote. That component is missing its required `endpoint`, so the agent fails
validation and refuses to start. The user's actual override silently never applies, and
the error message points at a component they never knowingly created.

The generalizable lesson: **the component IDs in our bundled config are a public API.**
They are the join keys that user overrides bind to. Renaming one is a breaking change
regardless of whether the underlying OTel type still accepts the old spelling, and the
same breakage would occur if we renamed only the instance part (`otlphttp/observe` ->
`otlphttp/observe_exporter`).

## How other distributions handle this

Upstream provides a purpose-built hook for exactly this: the `confmap.Converter`
interface, documented as existing so "distributions ... build backwards compatible config
converters." The resolver's contract is:

1. Start with an empty result.
2. Retrieve each config URI and merge it into the result.
3. Resolve embedded URIs.
4. **Run each registered `Converter` over the merged result.**
5. Return the effective config.

Step 4 is the seam. A converter sees the fully merged config as a mutable
`*confmap.Conf` and can rewrite it before anything is unmarshalled into component
configs, so users' files never have to change.

**Splunk** leans on this the hardest. Their `internal/configconverter` package holds one
converter per historical breaking change (for example `loglevel_to_verbosity.go`). As
they describe it, the converter "runs as part of config initialization and translates the
confmap entries over, so it doesn't change files. It emits warnings when it finds
conversions to make, with a snippet showing the changes to make, so folks can apply them
themselves." This is how they moved users off `spanmetricsprocessor` onto
`spanmetricsconnector` — a far bigger change than a rename — without a flag day. The
pattern was good enough that they proposed upstreaming it as a shared `configconverter`
component.

**Datadog** ships the same idea as a first-class piece of DDOT, at
`comp/otelcol/converter`. Their framing is broader than migrations: it "enhances
user-provided configurations ... automatically checking for known misconfigurations to
reduce errors," and exposes an API returning **both** the original and the enhanced
config so operators can see what the agent did to their input. Datadog's need for this is
in fact why `confmap.Converter` support was added to the OTel Collector Builder manifest.

**New Relic** (NRDOT) takes the lighter-weight path: they renamed the nodes in their own
bundled configs to track upstream (`cumulativetodelta` -> `cumulative_to_delta` in
v2.0.0) and rely on the upstream type alias plus a release-note callout for user configs.
That works for them because NRDOT hands users a whole config file to edit rather than
deep-merging fragments into a bundled config, so they do not have our join-key problem.
It is the approach we effectively took, and it is the one that broke.

Upstream's own coding guidelines codify the expectation on the component side: renamed
components SHOULD add the `lower_snake_case` name as primary, MAY keep the old name as a
deprecated alias, and MUST document the migration path. Aliases are expected to live
"1 or even 2 years."

## What we do

A single `confmap.Converter`, implemented in
[internal/configconverter](../internal/configconverter).

### The converter

We construct `ResolverSettings` ourselves, so this needed no OCB changes.
`confmap.ResolverSettings` exposes `ConverterFactories` alongside `ProviderFactories`:

```go
ConverterFactories: []confmap.ConverterFactory{
	configconverter.NewFactory(),
},
ConverterSettings: confmap.ConverterSettings{Logger: logger},
```

This is a strict superset of what a pre-merge rewrite of `otel_config_overrides` would
catch, since it also covers legacy IDs arriving via `--config` files and `--set` flags.
Both `--render-otel` code paths in `internal/commands/config/printers.go` build off the
same `ResolverSettings`, so they display the converted config for free.

Three design details that matter:

**Rewrite at leaf granularity, not whole blocks.** Running post-merge means the converter
has to combine the legacy and canonical blocks itself, which sounds like a place to
diverge from confmap's real semantics around nulls and list replacement. Rewriting each
flat leaf key from `Conf.AllKeys()` and re-merging sidesteps that entirely: a legacy block
and an existing canonical block combine as a true deep merge, with the legacy leaves
winning. That is the intended precedence, since the bundled config is canonical by
construction and anything found under a legacy ID was authored by the user.

For lists (pipeline references and `service::extensions`) the converter deletes the key
before merging the rewritten list. A plain merge would *append* rather than replace when
the `confmap.enableMergeAppendOption` feature gate is on.

**Key the table on full component IDs, not bare type names.** A user's own
`filelog/mycustom` is not broken by anything we did; the upstream type alias resolves it
fine. The breakage is specific to IDs that collide with our bundled config, so a table of
`otlphttp/observe -> otlp_http/observe` rather than `otlphttp -> otlp_http` fixes exactly
what we broke and leaves user-owned components untouched.

**Warn rather than silently fix.** Following Splunk, each applied mapping logs the exact
old and new ID once, so the deprecation surfaces in the user's logs and they eventually
migrate.

One known limitation: `Conf.Get` sanitizes `ExpandedValue` wrappers and the unsanitized
accessor is collector-internal. Env expansion already ran by the time converters execute,
so moved values are correct; they just lose the original `${env:...}` text used for
redaction metadata. This only affects leaves a user wrote under a legacy name.

### The compat table

The converter reads a single `map[string]string` of legacy -> canonical ID in
[internal/configconverter/mappings.go](../internal/configconverter/mappings.go). That
table is the artifact that makes the policy below enforceable.

### Not yet built: the CI guard

The guard that would have caught the original break before release: we already snapshot
the fully rendered otel config under `internal/commands/config/test/snap*-*-output.yaml`.
Extracting the set of component IDs from those snapshots into a checked-in list, and
failing CI when an ID disappears from it without a corresponding entry in the compat
table, would turn "did anyone remember this is a breaking change?" from a review question
into a build failure. Until that exists, the policy below is enforced by review alone.

## Policy going forward

Treat every component ID that appears in a rendered config as public API.

- Any PR that renames or removes a bundled component ID adds its compat-table entry in
  the same PR. Enforced by review today; see the CI guard note above.
- Keep entries for at least as long as upstream keeps its own aliases — a year is a
  reasonable floor, and there is little cost to keeping them longer.
- Log a warning naming the old and new ID when a mapping fires, so the deprecation is
  visible in the user's logs rather than only in release notes.
- Removing a compat-table entry is itself a breaking change and needs its own release
  note.
- Ship the release note anyway. The converter buys users time; it is not a substitute for
  telling them.

One thing worth flagging as a longer-term concern: this whole class of problem exists
because `otel_config_overrides` makes our internal component IDs part of the user
contract. The compat table manages that contract; it does not remove it. If override
usage grows, the more durable fix is to expose higher-level knobs for the common cases
(queue sizing, endpoints, filtering) so fewer users need to name our components at all.

## References

- [`confmap.Converter`](https://github.com/open-telemetry/opentelemetry-collector/blob/main/confmap/converter.go) and the [confmap README](https://github.com/open-telemetry/opentelemetry-collector/blob/main/confmap/README.md) resolver ordering
- [Splunk `internal/configconverter`](https://github.com/signalfx/splunk-otel-collector/tree/main/internal/configconverter) and the [proposal to upstream it](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues/30654)
- [Datadog `comp/otelcol/converter`](https://github.com/DataDog/datadog-agent/tree/main/comp/otelcol/converter)
- [Component naming convention discussion](https://github.com/open-telemetry/opentelemetry-collector/issues/14208) and the [core rename tracking issue](https://github.com/open-telemetry/opentelemetry-collector/issues/14396)
- [Upstream coding guidelines on renames and deprecation](https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/coding-guidelines.md)
