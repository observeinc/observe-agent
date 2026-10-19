# OpenTelemetry Collector examples

These examples show how to send telemetry to Observe with a plain [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/) (`otelcol-contrib`) instead of the Observe Agent. They use the same processors and pipelines as the Observe Agent, so the data arrives in the shape that Observe's explorers expect.

| Example | Use it for |
| :------ | :--------- |
| [`collector/host.yaml`](collector/host.yaml) | A collector on a Linux host: host metrics, host logs from `/var/log` and journald, application traces, logs, and metrics over OTLP, RED (request rate, error, and duration) metrics generated from traces, and the collector's own internal metrics. On Windows and macOS, remove the receivers and `hostmetrics` scrapers your platform doesn't support, such as journald and `processes` |
| [Kubernetes example](https://github.com/observeinc/helm-charts/tree/main/charts/agent/examples/oss-collector) | Collectors running in Kubernetes with the OpenTelemetry Collector Helm chart, in the `observeinc/helm-charts` repository |

## Run the host example

1. Create an ingest token in Observe. See [Configure your own OTel collector without Kubernetes](https://docs.observeinc.com/docs/configure-your-own-otel-collector-in-a-non-kubernetes-environment#/).
2. Set the environment variables that the configuration reads. All three are required; the collector doesn't start if one is missing.

   ```sh
   export OBSERVE_COLLECTION_URL=https://123456789012.collect.observeinc.com
   export OBSERVE_TOKEN=<your ingest token>
   export DEPLOYMENT_ENVIRONMENT=prod
   ```

   `DEPLOYMENT_ENVIRONMENT` is the environment for telemetry that doesn't set its own: telemetry that already sets `deployment.environment.name` (or the deprecated `deployment.environment`) keeps its value. It is still required, even if all your applications set their own environment. See [Set the deployment environment](https://docs.observeinc.com/docs/set-the-deployment-environment#/).

3. Start the collector with the `otelcol-contrib` version named at the top of `collector/host.yaml`, the collector version the Observe Agent is built with:

   ```sh
   otelcol-contrib --config collector/host.yaml
   ```

The OTLP receiver listens on `localhost:4317` (gRPC) and `localhost:4318` (HTTP). To receive telemetry from other machines, change the endpoints to `0.0.0.0`.

The collector also needs:

- Read access to `/var/log` and to the systemd journal, and the `journalctl` binary, for host logs. Running it as root, or as a user in the `adm` and `systemd-journal` groups, gives it both.
- A writable `/var/lib/otelcol/file_storage`, where it keeps its log read positions across restarts. It creates the directory if it can.

## Where these files come from

The files in `collector/` are generated from the Observe Agent's configuration and checked in CI, so don't edit them by hand. See [`scripts/generate_oss_examples`](../../scripts/generate_oss_examples/).
