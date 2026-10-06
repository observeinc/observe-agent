# OpenTelemetry Collector examples

These examples show how to send telemetry to Observe with a plain [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) (`otelcol-contrib`) instead of the Observe Agent. They use the same processors and pipelines as the Observe Agent, so the data arrives in the shape that Observe's explorers expect.

| Example | Use it for |
| :------ | :--------- |
| [`host/collector.yaml`](host/collector.yaml) | A collector on a Linux host: application traces, logs, and metrics over OTLP, RED (request rate, error, and duration) metrics generated from traces, and host metrics. On Windows and macOS, remove the `hostmetrics` scrapers your platform doesn't support, such as `processes` on Windows |
| [Kubernetes example](https://github.com/observeinc/helm-charts/tree/main/examples/oss-collector) | A collector running in Kubernetes, in the `observeinc/helm-charts` repository |

## Run the host example

1. Create an ingest token in Observe. See [Configure your own OTel collector without Kubernetes](https://docs.observeinc.com/docs/configure-your-own-otel-collector-in-a-non-kubernetes-environment#/).
2. Set the environment variables that the configuration reads:

   ```sh
   export OBSERVE_COLLECTION_URL=https://123456789012.collect.observeinc.com
   export OBSERVE_TOKEN=<your ingest token>
   export DEPLOYMENT_ENVIRONMENT=prod
   ```

   `DEPLOYMENT_ENVIRONMENT` is a fallback: telemetry that already sets `deployment.environment.name` (or the deprecated `deployment.environment`) keeps its own value. See [Set the deployment environment](https://docs.observeinc.com/docs/set-the-deployment-environment#/).

3. Start the collector:

   ```sh
   otelcol-contrib --config host/collector.yaml
   ```

The OTLP receiver listens on `localhost:4317` (gRPC) and `localhost:4318` (HTTP). To receive telemetry from other machines, change the endpoints to `0.0.0.0`.

## How these examples stay up to date

`host/collector.yaml` is generated, so don't edit it by hand. `make generate-oss-examples` renders the Observe Agent with [`host/agent-reference-config.yaml`](host/agent-reference-config.yaml) and removes the parts that only the agent needs, such as its self-monitoring pipelines and persistent queue storage. See [`scripts/generate_oss_examples`](../../scripts/generate_oss_examples/main.go) for the exact list.

These checks run in CI:

- `code-gen-check` regenerates the examples and fails if the committed files differ, so a change to the agent's bundled configuration also updates the examples.
- `Test_OSSHostExample` runs the example, sends traces, logs, and metrics, and checks the deployment environment on everything it forwards, including RED metrics.
- The `OSS collector examples` workflow validates the examples with upstream `otelcol-contrib`: on pull requests with the collector version the agent is built with, and every week with the latest release.

To change what the example contains, change the agent's bundled configuration or `agent-reference-config.yaml`, then run `make generate-oss-examples`.
