# Delta to cumulative processor

<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [alpha]: metrics   |
| Distributions | [contrib], [k8s] |
| Warnings      | [Statefulness](#warnings) |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Aprocessor%2Fdeltatocumulative%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Aprocessor%2Fdeltatocumulative) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Aprocessor%2Fdeltatocumulative%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Aprocessor%2Fdeltatocumulative) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@sh0rez](https://www.github.com/sh0rez), [@RichieSams](https://www.github.com/RichieSams), [@jpkrohling](https://www.github.com/jpkrohling) |

[alpha]: https://github.com/open-telemetry/opentelemetry-collector/blob/main/docs/component-stability.md#alpha
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
[k8s]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-k8s
<!-- end autogenerated section -->


## Description

The delta to cumulative processor (`deltatocumulativeprocessor`) converts
metrics from delta temporality to cumulative, by accumulating samples in memory.

## Configuration

``` yaml
processors:
    deltatocumulative:
        # how long until a series not receiving new samples is removed
        [ max_stale: <duration> | default = 5m ]
 
        # upper limit of streams to track. new streams exceeding this limit
        # will be dropped
        [ max_streams: <int> | default = 9223372036854775807 (max int) ]

```

There is no further configuration required. All delta samples are converted to cumulative.

## Troubleshooting

When [Telemetry is
enabled](https://opentelemetry.io/docs/collector/configuration/#telemetry), this component exports [several metrics](./documentation.md). 