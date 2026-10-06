package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

var testReference = func() referenceValues {
	ref := referenceValues{ObserveURL: "https://123456.collect.observeinc.com/", Token: "ds:secret"}
	ref.ResourceEnv.Name = "oss-example-environment"
	return ref
}()

const testRendered = `
connectors:
  count: {}
receivers:
  otlp: {}
  filestats/agent: {}
processors:
  batch: {}
  resource/agent_instance: {}
  resource/observe_global_resource_attributes:
    attributes:
      - key: deployment.environment.name
        action: insert
        value: oss-example-environment
exporters:
  otlphttp/observe:
    endpoint: https://123456.collect.observeinc.com/v2/otel
    headers:
      authorization: Bearer ds:secret
  debug: {}
extensions:
  file_storage: {}
service:
  extensions: [file_storage]
  pipelines:
    logs/forward:
      receivers: [otlp]
      processors: [resource/agent_instance, resource/observe_global_resource_attributes, batch]
      exporters: [otlphttp/observe, count]
    metrics/agent-filestats:
      receivers: [filestats/agent]
      processors: [batch]
      exporters: [otlphttp/observe]
`

func Test_generate(t *testing.T) {
	out, err := generate([]byte(testRendered), testReference)
	require.NoError(t, err)

	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(out, &cfg))

	assert.NotContains(t, cfg, "connectors")
	assert.NotContains(t, cfg, "extensions")
	assert.Equal(t, map[string]any{
		"pipelines": map[string]any{
			"logs/forward": map[string]any{
				"receivers":  []any{"otlp"},
				"processors": []any{"resource/observe_global_resource_attributes", "batch"},
				"exporters":  []any{"otlphttp/observe"},
			},
		},
	}, cfg["service"], "agent-only components and pipelines are removed")

	exporter := cfg["exporters"].(map[string]any)["otlphttp/observe"].(map[string]any)
	assert.Equal(t, "${env:OBSERVE_COLLECTION_URL}/v2/otel", exporter["endpoint"])
	assert.Equal(t, "Bearer ${env:OBSERVE_TOKEN}", exporter["headers"].(map[string]any)["authorization"])

	attrs := cfg["processors"].(map[string]any)["resource/observe_global_resource_attributes"].(map[string]any)["attributes"].([]any)
	assert.Equal(t, "${env:DEPLOYMENT_ENVIRONMENT}", attrs[0].(map[string]any)["value"])
}

func Test_generateRequiresReferenceValues(t *testing.T) {
	_, err := generate([]byte(testRendered), referenceValues{ObserveURL: "https://x", Token: "ds:secret"})
	require.Error(t, err)
}

func Test_generateRejectsTokenInKeys(t *testing.T) {
	// Values are rewritten, keys are not; the final check must catch a token that survives.
	_, err := generate([]byte(testRendered+"\nds:secret: leaked\n"), testReference)
	require.Error(t, err)
}
