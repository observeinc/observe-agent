# Observe Agent

Code for the Observe agent and CLI. The agent code is based on the OpenTelemetry Collector. 

# Build

To run the code you need to have `golang v1.22.7` installed. Then you can run the following command to compile the binary.

```
go build -o observe-agent
```

## Adding new components

Before adding new components, you'll need to install the [Otel Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) tool. If you're running on mac and arm64 (M chips) you can run the following command

```
make install-ocb
```

Otherwise, see instructions to install at https://opentelemetry.io/docs/collector/custom-collector/#step-1---install-the-builder

To add new components, you can modify the `builder-config.yaml` file. Add the component to the correct section and then run the following command.
```
make build-ocb
```

This command should add the new dependencies and code in the correct places. You can build the agent afterwards with `go build` to confirm. 

Afterwards, you should add the new component to the `Components` section below. 

## Running

To start the observe agent after building the binary run the following command. 

```
./observe-agent start
```

## Components

Current OTEL Collector Version: `v0.110.0`

This section lists the components that are included in the Observe Distribution of the OpenTelemetry Collector.

| Receivers                                                | Processors                                            | Exporters                                              | Extensions                           | Connectors                  |
|----------------------------------------------------------|-------------------------------------------------------|--------------------------------------------------------|--------------------------------------|-----------------------------|
| [awsecscontainermetrics][awsecscontainermetricsreceiver] | [attributes][attributesprocessor]                     | [debug][debugexporter]                                 | [file_storage][filestorage]          | [count][countconnector]     |
| [docker_stats][dockerstatsreceiver]                      | [batch][batchprocessor]                               | [file][fileexporter]                                   | [health_check][healthcheckextension] | [forward][forwardconnector] |
| [elasticsearch][elasticsearchreceiver]                   | [deltatocumulative][deltatocumulativeprocessor]       | [otlphttp][otlphttpexporter]                           | [zpages][zpagesextension]            |                             |
| [filelog][filelogreceiver]                               | [filter][filterprocessor]                             | [prometheusremotewrite][prometheusremotewriteexporter] |                                      |                             |
| [filestats][filestatsreceiver]                           | [k8sattributes][k8sattributesprocessor]               |                                                        |                                      |                             |
| [hostmetrics][hostmetricsreceiver]                       | [memory_limiter][memorylimiterprocessor]              |                                                        |                                      |                             |
| [iis][iisreceiver]                                       | [observek8sattributes][observek8sattributesprocessor] |                                                        |                                      |                             |
| [journald][journaldreceiver]                             | [probabilisticsampler][probabilisticsamplerprocessor] |                                                        |                                      |                             |
| [k8s_cluster][k8sclusterreceiver]                        | [redaction][redactionprocessor]                       |                                                        |                                      |                             |
| [k8sobjects][k8sobjectsreceiver]                         | [resource][resourceprocessor]                         |                                                        |                                      |                             |
| [kafkametrics][kafkametricsreceiver]                     | [resourcedetection][resourcedetectionprocessor]       |                                                        |                                      |                             |
| [kafka][kafkareceiver]                                   | [span][spanprocessor]                                 |                                                        |                                      |                             |
| [kubeletstats][kubeletstatsreceiver]                     | [tailsampling][tailsamplingprocessor]                 |                                                        |                                      |                             |
| [mongodb][mongodbreceiver]                               | [transform][transformprocessor]                       |                                                        |                                      |                             |
| [otlp][otlpreceiver]                                     |                                                       |                                                        |                                      |                             |
| [prometheus][prometheusreceiver]                         |                                                       |                                                        |                                      |                             |
| [redis][redisreceiver]                                   |                                                       |                                                        |                                      |                             |
| [statsd][statsdreceiver]                                 |                                                       |                                                        |                                      |                             |
| [tcplog][tcplogreceiver]                                 |                                                       |                                                        |                                      |                             |
| [udplog][udplogreceiver]                                 |                                                       |                                                        |                                      |                             |
| [windowseventlog][windowseventlogreceiver]               |                                                       |                                                        |                                      |                             |

[awsecscontainermetricsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/awsecscontainermetricsreceiver
[dockerstatsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/dockerstatsreceiver
[elasticsearchreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/elasticsearchreceiver
[filelogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/filelogreceiver
[filestatsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/filestatsreceiver
[hostmetricsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/hostmetricsreceiver
[iisreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/iisreceiver
[journaldreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/journaldreceiver
[k8sclusterreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/k8sclusterreceiver
[k8sobjectsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/k8sobjectsreceiver
[kafkametricsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/kafkametricsreceiver
[kafkareceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/kafkareceiver
[kubeletstatsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/kubeletstatsreceiver
[mongodbreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/mongodbreceiver
[otlpreceiver]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.110.0/receiver/otlpreceiver
[prometheusreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/prometheusreceiver
[redisreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.102.0/receiver/redisreceiver
[statsdreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/statsdreceiver
[tcplogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/tcplogreceiver
[udplogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/udplogreceiver
[windowseventlogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/receiver/windowseventlogreceiver
[attributesprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/attributesprocessor
[batchprocessor]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.110.0/processor/batchprocessor
[deltatocumulativeprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/deltatocumulativeprocessor
[filterprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/filterprocessor
[k8sattributesprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/k8sattributesprocessor
[memorylimiterprocessor]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.110.0/processor/memorylimiterprocessor
[observek8sattributesprocessor]: ./components/processors/observek8sattributesprocessor
[probabilisticsamplerprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/probabilisticsamplerprocessor
[redactionprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/redactionprocessor
[resourceprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/resourceprocessor
[resourcedetectionprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/resourcedetectionprocessor
[spanprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/spanprocessor
[tailsamplingprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/tailsamplingprocessor
[transformprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/processor/transformprocessor
[debugexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.110.0/exporter/debugexporter
[fileexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/exporter/fileexporter
[otlphttpexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.110.0/exporter/otlphttpexporter
[prometheusremotewriteexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/exporter/prometheusremotewriteexporter
[countconnector]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/connector/countconnector
[forwardconnector]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.110.0/connector/forwardconnector
[filestorage]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/extension/storage/filestorage
[healthcheckextension]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.110.0/extension/healthcheckextension
[zpagesextension]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.110.0/extension/zpagesextension