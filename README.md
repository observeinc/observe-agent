# Observe Agent

Code for the Observe agent and CLI. The agent code is based on the OpenTelemetry Collector.

## Configuration

See <https://docs.observeinc.com/en/latest/content/observe-agent/configuration.html> for observe-agent configuration documentation. We also offer a json schema for the observe-agent config file, which can be added to editors to provide autocomplete and validation. The schema can be found at <https://github.com/observeinc/observe-agent/releases/latest/download/observe-agent.schema.json>.

## Build

To run the code you need to have `golang v1.24.6` installed. Then you can run the following command to compile the binary.

```sh
go build -o observe-agent
```

## Installing local builds on Mac or Linux

To install a local build of the agent as a Mac Launch Daemon or Linux Systemd service, run the following commands. First, build the release snapshot:

```sh
goreleaser release --snapshot --clean --verbose --single-target
```

Then, run the agent install script, pointing it to the snapshot build:

```sh
# For Mac
ZIP_DIR=./dist/darwin_arm64_v8.0/observe-agent_Darwin_arm64.zip ./scripts/install_mac.sh --token <token> --observe_url <observe_url>
```

or

```sh
# For Linux
ZIP_DIR=./dist/linux_amd64_v1/observe-agent_Linux_x86_64.tar.gz ./scripts/install_linux.sh --token <token> --observe_url <observe_url>
```

## Adding new components

Before adding new components, you'll need to install the [Otel Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) tool. If you're running on mac and arm64 (M chips) you can run the following command

```sh
make install-ocb
```

Otherwise, see instructions to install at [https://opentelemetry.io/docs/collector/custom-collector/#step-1---install-the-builder]

To add new components, you can modify the `builder-config.yaml` file. Add the component to the correct section and then run the following command.

```sh
make build-ocb
```

This command should add the new dependencies and code in the correct places. You can build the agent afterwards with `go build` to confirm.

Afterwards, you should add the new component to the `Components` section below.

## Running

To start the observe agent after building the binary run the following command.

```sh
./observe-agent start
```

## Components

Current OTEL Collector Version: `v0.131.0`

This section lists the components that are included in the Observe Distribution of the OpenTelemetry Collector.

| Receivers                                                | Processors                                            | Exporters                                              | Extensions                              | Connectors                          |
|----------------------------------------------------------|-------------------------------------------------------|--------------------------------------------------------|-----------------------------------------|-------------------------------------|
| [awsecscontainermetrics][awsecscontainermetricsreceiver] | [attributes][attributesprocessor]                     | [debug][debugexporter]                                 | [cgroupruntime][cgroupruntimeextension] | [count][countconnector]             |
| [docker_stats][dockerstatsreceiver]                      | [batch][batchprocessor]                               | [file][fileexporter]                                   | [file_storage][filestorage]             | [forward][forwardconnector]         |
| [elasticsearch][elasticsearchreceiver]                   | [cumulativetodelta][cumulativetodeltaprocessor]       | [loadbalancing][loadbalancingexporter]                 | [health_check][healthcheckextension]    | [spanmetrics][spanmetricsconnector] |
| [filelog][filelogreceiver]                               | [deltatocumulative][deltatocumulativeprocessor]       | [nop][nopexporter]                                     | [pprof][pprofextension]                 |                                     |
| [filestats][filestatsreceiver]                           | [filter][filterprocessor]                             | [otlp][otlpexporter]                                   | [zpages][zpagesextension]               |                                     |
| [hostmetrics][hostmetricsreceiver]                       | [groupbyattrs][groupbyattrsprocessor]                 | [otlphttp][otlphttpexporter]                           |                                         |                                     |
| [httpcheck][httpcheckreceiver]                           | [k8sattributes][k8sattributesprocessor]               | [prometheusremotewrite][prometheusremotewriteexporter] |                                         |                                     |
| [iis][iisreceiver]                                       | [memory_limiter][memorylimiterprocessor]              |                                                        |                                         |                                     |
| [journald][journaldreceiver]                             | [metricstransform][metricstransformprocessor]         |                                                        |                                         |                                     |
| [k8s_cluster][k8sclusterreceiver]                        | [observek8sattributes][observek8sattributesprocessor] |                                                        |                                         |                                     |
| [k8sobjects][k8sobjectsreceiver]                         | [probabilisticsampler][probabilisticsamplerprocessor] |                                                        |                                         |                                     |
| [kafkametrics][kafkametricsreceiver]                     | [redaction][redactionprocessor]                       |                                                        |                                         |                                     |
| [kafka][kafkareceiver]                                   | [resource][resourceprocessor]                         |                                                        |                                         |                                     |
| [kubeletstats][kubeletstatsreceiver]                     | [resourcedetection][resourcedetectionprocessor]       |                                                        |                                         |                                     |
| [mongodb][mongodbreceiver]                               | [span][spanprocessor]                                 |                                                        |                                         |                                     |
| [nop][nopreceiver]                                       | [tailsampling][tailsamplingprocessor]                 |                                                        |                                         |                                     |
| [otlp][otlpreceiver]                                     | [transform][transformprocessor]                       |                                                        |                                         |                                     |
| [prometheus][prometheusreceiver]                         |                                                       |                                                        |                                         |                                     |
| [redis][redisreceiver]                                   |                                                       |                                                        |                                         |                                     |
| [snmp][snmpreceiver]                                     |                                                       |                                                        |                                         |                                     |
| [sqlquery][sqlqueryreceiver]                             |                                                       |                                                        |                                         |                                     |
| [statsd][statsdreceiver]                                 |                                                       |                                                        |                                         |                                     |
| [tcplog][tcplogreceiver]                                 |                                                       |                                                        |                                         |                                     |
| [udplog][udplogreceiver]                                 |                                                       |                                                        |                                         |                                     |
| [windowseventlog][windowseventlogreceiver]               |                                                       |                                                        |                                         |                                     |

[awsecscontainermetricsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/awsecscontainermetricsreceiver
[dockerstatsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/dockerstatsreceiver
[elasticsearchreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/elasticsearchreceiver
[filelogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/filelogreceiver
[filestatsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/filestatsreceiver
[hostmetricsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/hostmetricsreceiver
[httpcheckreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/httpcheckreceiver
[iisreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/iisreceiver
[journaldreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/journaldreceiver
[k8sclusterreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/k8sclusterreceiver
[k8sobjectsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/k8sobjectsreceiver
[kafkametricsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/kafkametricsreceiver
[kafkareceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/kafkareceiver
[kubeletstatsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/kubeletstatsreceiver
[mongodbreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/mongodbreceiver
[nopreceiver]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/receiver/nopreceiver
[otlpreceiver]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/receiver/otlpreceiver
[prometheusreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/prometheusreceiver
[redisreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/redisreceiver
[snmpreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/snmpreceiver
[sqlqueryreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/sqlqueryreceiver
[statsdreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/statsdreceiver
[tcplogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/tcplogreceiver
[udplogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/udplogreceiver
[windowseventlogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/receiver/windowseventlogreceiver
[attributesprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/attributesprocessor
[batchprocessor]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/processor/batchprocessor
[cumulativetodeltaprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/cumulativetodeltaprocessor
[deltatocumulativeprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/deltatocumulativeprocessor
[filterprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/filterprocessor
[groupbyattrsprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/groupbyattrsprocessor
[k8sattributesprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/k8sattributesprocessor
[memorylimiterprocessor]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/processor/memorylimiterprocessor
[metricstransformprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/metricstransformprocessor
[observek8sattributesprocessor]: ./components/processors/observek8sattributesprocessor
[probabilisticsamplerprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/probabilisticsamplerprocessor
[redactionprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/redactionprocessor
[resourceprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/resourceprocessor
[resourcedetectionprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/resourcedetectionprocessor
[spanprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/spanprocessor
[tailsamplingprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/tailsamplingprocessor
[transformprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/processor/transformprocessor
[debugexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/exporter/debugexporter
[fileexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/exporter/fileexporter
[loadbalancingexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/exporter/loadbalancingexporter
[nopexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/exporter/nopexporter
[otlpexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/exporter/otlpexporter
[otlphttpexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/exporter/otlphttpexporter
[prometheusremotewriteexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/exporter/prometheusremotewriteexporter
[countconnector]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/connector/countconnector
[forwardconnector]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/connector/forwardconnector
[spanmetricsconnector]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/connector/spanmetricsconnector
[filestorage]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/extension/storage/filestorage
[cgroupruntimeextension]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/extension/cgroupruntimeextension
[healthcheckextension]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/extension/healthcheckextension
[pprofextension]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.131.0/extension/pprofextension
[zpagesextension]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.131.0/extension/zpagesextension
