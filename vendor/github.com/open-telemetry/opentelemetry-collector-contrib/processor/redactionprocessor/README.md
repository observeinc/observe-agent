# Redaction processor

<!-- status autogenerated section -->
| Status        |           |
| ------------- |-----------|
| Stability     | [beta]: traces   |
| Distributions | [contrib] |
| Issues        | [![Open issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aopen%20label%3Aprocessor%2Fredaction%20&label=open&color=orange&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aopen+is%3Aissue+label%3Aprocessor%2Fredaction) [![Closed issues](https://img.shields.io/github/issues-search/open-telemetry/opentelemetry-collector-contrib?query=is%3Aissue%20is%3Aclosed%20label%3Aprocessor%2Fredaction%20&label=closed&color=blue&logo=opentelemetry)](https://github.com/open-telemetry/opentelemetry-collector-contrib/issues?q=is%3Aclosed+is%3Aissue+label%3Aprocessor%2Fredaction) |
| [Code Owners](https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CONTRIBUTING.md#becoming-a-code-owner)    | [@dmitryax](https://www.github.com/dmitryax), [@mx-psi](https://www.github.com/mx-psi), [@TylerHelmuth](https://www.github.com/TylerHelmuth) |
| Emeritus      | [@leonsp-ai](https://www.github.com/leonsp-ai) |

[beta]: https://github.com/open-telemetry/opentelemetry-collector#beta
[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
<!-- end autogenerated section -->

This processor deletes span attributes that don't match a list of allowed span
attributes. It also masks span attribute values that match a blocked value
list. Span attributes that aren't on the allowed list are removed before any
value checks are done.

## Use Cases

Typical use-cases:

* Prevent sensitive fields from accidentally leaking into traces
* Ensure compliance with legal, privacy, or security requirements

For example:

* EU General Data Protection Regulation (GDPR) prohibits the transfer of any
  personal data like birthdates, addresses, or ip addresses across borders
  without explicit consent from the data subject. Popular trace aggregation
  services are located in US, not in EU. You can use the redaction processor
  to scrub personal data from your data.
* PRC legislation prohibits the transfer of geographic coordinates outside of
  the PRC. Popular trace aggregation services are located in US, not in the
  PRC. You can use the redaction processor to scrub geographic coordinates
  from your data.
* Payment Card Industry (PCI) Data Security Standards prohibit logging certain
  things or storing them unencrypted. You can use the redaction processor to
  scrub them from your traces.

The above is written by an engineer, not a lawyer. The redaction processor is
intended as one line of defence rather than the only compliance measure in
place.

## Processor Configuration

Please refer to [config.go](./config.go) for the config spec.

Examples:

```yaml
processors:
  redaction:
    # allow_all_keys is a flag which when set to true, which can disables the
    # allowed_keys list. The list of blocked_values is applied regardless. If
    # you just want to block values, set this to true.
    allow_all_keys: false
    # allowed_keys is a list of span attribute keys that are kept on the span and
    # processed. The list is designed to fail closed. If allowed_keys is empty,
    # no span attributes are allowed and all span attributes are removed. To
    # allow all keys, set allow_all_keys to true.
    allowed_keys:
      - description
      - group
      - id
      - name
    # Ignore the following attributes, allow them to pass without redaction.
    # Any keys in this list are allowed so they don't need to be in both lists.
    ignored_keys:
      - safe_attribute
    # blocked_values is a list of regular expressions for blocking values of
    # allowed span attributes. Values that match are masked
    blocked_values:
      - "4[0-9]{12}(?:[0-9]{3})?" ## Visa credit card number
      - "(5[1-5][0-9]{14})"       ## MasterCard number
    # summary controls the verbosity level of the diagnostic attributes that
    # the processor adds to the spans when it redacts or masks other
    # attributes. In some contexts a list of redacted attributes leaks
    # information, while it is valuable when integrating and testing a new
    # configuration. Possible values:
    # - `debug` includes both redacted key counts and names in the summary
    # - `info` includes just the redacted key counts in the summary
    # - `silent` omits the summary attributes
    summary: debug
```

Refer to [config.yaml](./testdata/config.yaml) for how to fit the configuration
into an OpenTelemetry Collector pipeline definition.

Ignored attributes are processed first so they're always allowed and never
blocked. This field should only be used where you know the data is always
safe to send to the telemetry system.

Only span attributes included on the list of allowed keys list are retained.
If `allowed_keys` is empty, then no span attributes are allowed. All span
attributes are removed in that case. To keep all span attributes, you should
explicitly set `allow_all_keys` to true.

`blocked_values` applies to the values of the allowed keys. If the value of an
allowed key matches the regular expression for a blocked value, the matching
part of the value is then masked with a fixed length of asterisks.

For example, if `notes` is on the list of allowed keys, then the `notes` span
attribute is retained. However, if there is a value such as a credit card
number in the `notes` field that matched a regular expression on the list of
blocked values, then that value is masked.