# Observe Agent

Code for the Observe agent and CLI. The agent code is based on the OpenTelemetry Collector. 

# Build

To run the code you need to have `golang v1.21.7` installed. Then you can run the following command to compile the binary.

```
go build -o observe-agent
```


## Running

To start the observe agent after building the binary run the following command. 

```
./observe-agent start
```

## Components

This section lists the components that are included in the Observe Distribution of the OpenTelemetry Collector.


| Receivers                                  | Processors                                      | Exporters                    | Extensions                           | Connectors              |
|--------------------------------------------|-------------------------------------------------|------------------------------|--------------------------------------|-------------------------|
| [filelog][filelogreceiver]                 | [attributes][attributesprocessor]               | [debug][debugexporter]       | [file_storage][filestorage]          | [count][countconnector] |
| [filestats][filestatsreceiver]             | [batch][batchprocessor]                         | [file][fileexporter]         | [health_check][healthcheckextension] |                         |
| [hostmetrics][hostmetricsreceiver]         | [memory_limiter][memorylimiterprocessor]        | [otlphttp][otlphttpexporter] | [zpages][zpagesextension]            |                         |
| [iis][iisreceiver]                         | [resourcedetection][resourcedetectionprocessor] |                              |                                      |                         |
| [journald][journaldreceiver]               | [transform][transformprocessor]                 |                              |                                      |                         |
| [otlp][otlpreceiver]                       |                                                 |                              |                                      |                         |
| [prometheus][prometheusreceiver]           |                                                 |                              |                                      |                         |
| [windowseventlog][windowseventlogreceiver] |                                                 |                              |                                      |                         |
|                                            |                                                 |                              |                                      |                         |
|                                            |                                                 |                              |                                      |                         |

[filelogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/receiver/filelogreceiver
[filestatsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/receiver/filestatsreceiver
[hostmetricsreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/receiver/hostmetricsreceiver
[iisreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/receiver/iisreceiver
[journaldreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/receiver/journaldreceiver
[otlpreceiver]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.103.0/receiver/otlpreceiver
[prometheusreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/receiver/prometheusreceiver
[windowseventlogreceiver]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/receiver/windowseventlogreceiver
[attributesprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/processor/attributesprocessor
[batchprocessor]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.103.0/processor/batchprocessor
[memorylimiterprocessor]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.103.0/processor/memorylimiterprocessor
[resourcedetectionprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/processor/resourcedetectionprocessor
[transformprocessor]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/processor/transformprocessor
[debugexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.103.0/exporter/debugexporter
[fileexporter]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/exporter/fileexporter
[otlphttpexporter]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.103.0/exporter/otlphttpexporter
[countconnector]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/connector/countconnector
[filestorage]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/extension/storage/filestorage
[healthcheckextension]: https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/v0.103.0/extension/healthcheckextension
[zpagesextension]: https://github.com/open-telemetry/opentelemetry-collector/tree/v0.103.0/extension/zpagesextension