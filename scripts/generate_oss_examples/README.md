# OSS collector example generator

This generator writes the OpenTelemetry Collector examples in [`examples/oss-collector`](../../examples/oss-collector/) from the Observe Agent's bundled configuration. Customers use those files; this directory is only how they're made.

`make generate-oss-examples` runs [`main.go`](main.go), which writes `examples/oss-collector/collector/host.yaml`:

1. Renders the collector configuration the Observe Agent would run with [`host/agent-reference-config.yaml`](host/agent-reference-config.yaml), with [`host/otel-config.yaml`](host/otel-config.yaml) merged on top. The otel config sets the exporter endpoints and token to `${env:...}` references.
2. Keeps only the pipelines listed in the `Makefile` target, and deletes every receiver, processor, exporter, connector, and extension those pipelines don't use. Agent-only parts such as the self-monitoring pipelines and persistent queue storage drop out this way.
3. Fails without writing the file if the result still contains the reference collection URL or token.

These checks run in CI:

- `code-gen-check` regenerates the examples and fails if the committed files differ, so a change to the agent's bundled configuration also updates the examples.
- `Test_DeploymentEnvironmentPrecedence` checks how the agent's configuration, which the example is generated from, handles the deployment environment on traces, logs, metrics, and RED metrics.
- The `OSS collector examples` workflow validates the examples with upstream `otelcol-contrib`: on pull requests with the collector version the agent is built with, and every week with the latest release.

To change what the example contains, change the agent's bundled configuration, the reference configs, or the pipelines in the `Makefile` target, then run `make generate-oss-examples`.
