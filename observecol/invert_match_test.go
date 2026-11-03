package observecol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/yaml.v3"
)

func mapFromYAML(t *testing.T, content string) map[string]any {
	var m map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(content), &m))
	return m
}

const errorsPolicy = `{name: errors, type: status_code, status_code: {status_codes: [ERROR]}}`
const healthzInverted = `{name: no-healthz, type: string_attribute, string_attribute: {key: http.target, values: [/healthz], invert_match: true}}`

func TestReliesOnInvertMatch(t *testing.T) {
	tests := []struct {
		name     string
		policies string
		want     bool
	}{
		{"inverted next to another policy", `[` + errorsPolicy + `, ` + healthzInverted + `]`, true},
		{"two inverted policies", `[` + healthzInverted + `, {name: slow, type: numeric_attribute, numeric_attribute: {key: duration, min_value: 1, max_value: 2, invert_match: true}}]`, true},
		{"lone inverted policy", `[` + healthzInverted + `]`, false},
		{"lone inverted policy and a drop policy", `[` + healthzInverted + `, {name: drop-debug, type: drop, drop: {drop_sub_policy: [{name: debug, type: boolean_attribute, boolean_attribute: {key: debug, value: true}}]}}]`, false},
		{"inverted inside and", `[` + errorsPolicy + `, {name: and, type: and, and: {and_sub_policy: [` + healthzInverted + `]}}]`, false},
		{"invert_match false", `[` + errorsPolicy + `, {name: healthz, type: string_attribute, string_attribute: {key: http.target, values: [/healthz], invert_match: false}}]`, false},
		{"no invert_match", `[` + errorsPolicy + `]`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, reliesOnInvertMatch(mapFromYAML(t, "policies: "+tt.policies)))
		})
	}
}

func TestPlanInvertMatchSwaps(t *testing.T) {
	conf := mapFromYAML(t, `
processors:
  tail_sampling/observe:
    policies: [`+errorsPolicy+`, `+healthzInverted+`]
  tail_sampling/sharded:
    num_shards: 2
    policies: [`+errorsPolicy+`, `+healthzInverted+`]
  tail_sampling/clean:
    policies: [`+errorsPolicy+`]
  tail_sampling/taken:
    policies: [`+errorsPolicy+`, `+healthzInverted+`]
  tailsampling_legacy/taken:
    policies: [`+errorsPolicy+`]
  filter/invert_match:
    invert_match: true
`)
	swaps := planInvertMatchSwaps(conf)
	require.Len(t, swaps, 3)

	assert.Equal(t, "tail_sampling/observe", swaps[0].Processor)
	assert.Equal(t, "tailsampling_legacy/observe", swaps[0].LegacyID)
	assert.NoError(t, swaps[0].Err)

	assert.Equal(t, "tail_sampling/sharded", swaps[1].Processor)
	assert.ErrorContains(t, swaps[1].Err, "num_shards")

	assert.Equal(t, "tail_sampling/taken", swaps[2].Processor)
	assert.ErrorContains(t, swaps[2].Err, "tailsampling_legacy/taken is already defined")
}

func TestInvertMatchConverterSwapsProcessor(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	converter := newInvertMatchConverterFactory().Create(confmap.ConverterSettings{Logger: zap.New(core)})
	conf := confmap.NewFromStringMap(mapFromYAML(t, `
processors:
  batch: {}
  tail_sampling:
    policies: [`+errorsPolicy+`, `+healthzInverted+`]
  tail_sampling/clean:
    policies: [`+errorsPolicy+`]
service:
  pipelines:
    traces:
      processors: [tail_sampling, batch]
    traces/clean:
      processors: [tail_sampling/clean, batch]
`))
	require.NoError(t, converter.Convert(context.Background(), conf))

	processors := conf.Get("processors").(map[string]any)
	assert.Contains(t, processors, "tailsampling_legacy")
	assert.NotContains(t, processors, "tail_sampling")
	assert.Contains(t, processors, "tail_sampling/clean")
	assert.Equal(t, []any{"tailsampling_legacy", "batch"}, conf.Get("service::pipelines::traces::processors"))
	assert.Equal(t, []any{"tail_sampling/clean", "batch"}, conf.Get("service::pipelines::traces/clean::processors"))

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Message, "runs it as tailsampling_legacy")
	assert.Equal(t, "tail_sampling", entries[0].ContextMap()["processor"])
}

func TestInvertMatchConverterKeepsUnsupportedConfig(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	converter := newInvertMatchConverterFactory().Create(confmap.ConverterSettings{Logger: zap.New(core)})
	conf := confmap.NewFromStringMap(mapFromYAML(t, `
processors:
  tail_sampling:
    num_shards: 2
    policies: [`+errorsPolicy+`, `+healthzInverted+`]
service:
  pipelines:
    traces:
      processors: [tail_sampling]
`))
	require.NoError(t, converter.Convert(context.Background(), conf))

	assert.Contains(t, conf.Get("processors").(map[string]any), "tail_sampling")
	assert.Equal(t, []any{"tail_sampling"}, conf.Get("service::pipelines::traces::processors"))

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Contains(t, entries[0].Message, "no longer exclude traces")
}
