package observecol

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
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

const upcomingChangesConfig = `
processors:
  batch: {}
  tail_sampling/observe:
    policies: [` + errorsPolicy + `, ` + healthzInverted + `]
  tail_sampling:
    policies: [` + errorsPolicy + `, ` + healthzInverted + `]
  tail_sampling/clean:
    policies: [` + errorsPolicy + `]
  filter/invert_match:
    invert_match: true
service:
  pipelines:
    traces:
      processors: [tail_sampling, batch]
`

func TestUpcomingChanges(t *testing.T) {
	changes := upcomingChanges(mapFromYAML(t, upcomingChangesConfig))
	require.Len(t, changes, 2)
	assert.Equal(t, "tail_sampling", changes[0].Component)
	assert.Equal(t, "tail_sampling/observe", changes[1].Component)
	assert.Contains(t, changes[0].Message, "no longer exclude traces")
}

func TestUpcomingChangesForReceivers(t *testing.T) {
	conf := mapFromYAML(t, `
receivers:
  otlp: {}
  http_check: {}
  httpcheck/legacy: {}
  mongodb: {}
  sqlserver/main: {}
  postgresql: {}
  hostmetrics/defaults:
    scrapers:
      cpu:
  hostmetrics/keeps_cpu:
    scrapers:
      cpu:
        metrics:
          system.cpu.time:
            attributes: [cpu, state]
  hostmetrics/utilization_without_cpu:
    scrapers:
      cpu:
        metrics:
          system.cpu.time:
            attributes: [cpu, state]
          system.cpu.utilization:
            enabled: true
  hostmetrics/time_disabled:
    scrapers:
      cpu:
        metrics:
          system.cpu.time:
            enabled: false
  hostmetrics/no_cpu:
    scrapers:
      memory:
`)
	got := map[string]string{}
	for _, change := range upcomingChanges(conf) {
		got[change.Component] = change.Message
	}
	assert.Equal(t, map[string]string{
		"http_check":                          httpCheckMessage,
		"httpcheck/legacy":                    httpCheckMessage,
		"mongodb":                             mongoDBMessage,
		"sqlserver/main":                      sqlServerMessage,
		"postgresql":                          postgreSQLMessage,
		"hostmetrics/defaults":                hostMetricsCPUMessage,
		"hostmetrics/utilization_without_cpu": hostMetricsCPUMessage,
	}, got)
}

func TestUpcomingChangesConverterOnlyWarns(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	converter := newUpcomingChangesConverterFactory().Create(confmap.ConverterSettings{Logger: zap.New(core)})
	conf := confmap.NewFromStringMap(mapFromYAML(t, upcomingChangesConfig))
	before := conf.ToStringMap()

	require.NoError(t, converter.Convert(context.Background(), conf))

	assert.Equal(t, before, conf.ToStringMap())
	entries := logs.All()
	require.Len(t, entries, 2)
	assert.Equal(t, "tail_sampling", entries[0].ContextMap()["component"])
	assert.Equal(t, "tail_sampling/observe", entries[1].ContextMap()["component"])
}

func TestUpcomingChangesResolvesSettings(t *testing.T) {
	settings := generateCollectorSettings([]string{"yaml:" + upcomingChangesConfig})
	changes, err := UpcomingChanges(context.Background(), settings)
	require.NoError(t, err)
	require.Len(t, changes, 2)
}

func newTrace(id byte, target string, status ptrace.StatusCode) ptrace.Traces {
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID{id})
	span.SetSpanID(pcommon.SpanID{id})
	span.Attributes().PutStr("http.target", target)
	span.Status().SetCode(status)
	return td
}

// sampledTraceIDs runs a /healthz error, an /api error and an /api OK trace through a tail_sampling
// processor that samples errors and uses invert_match to exclude /healthz, and returns the IDs it
// samples.
func sampledTraceIDs(t *testing.T) []pcommon.TraceID {
	factory := tailsamplingprocessor.NewFactory()
	cfg := factory.CreateDefaultConfig()
	policies := mapFromYAML(t, "policies: ["+errorsPolicy+", "+healthzInverted+"]")
	require.NoError(t, confmap.NewFromStringMap(policies).Unmarshal(cfg))

	sink := new(consumertest.TracesSink)
	proc, err := factory.CreateTraces(context.Background(), processortest.NewNopSettings(factory.Type()), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, proc.Start(context.Background(), componenttest.NewNopHost()))
	require.NoError(t, proc.ConsumeTraces(context.Background(), newTrace(1, "/healthz", ptrace.StatusCodeError)))
	require.NoError(t, proc.ConsumeTraces(context.Background(), newTrace(2, "/api", ptrace.StatusCodeError)))
	require.NoError(t, proc.ConsumeTraces(context.Background(), newTrace(3, "/api", ptrace.StatusCodeOk)))
	require.NoError(t, proc.Shutdown(context.Background()))

	var sampled []pcommon.TraceID
	for _, td := range sink.AllTraces() {
		sampled = append(sampled, td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).TraceID())
	}
	return sampled
}

// The agent keeps disableinvertdecisions off, so invert_match excludes the /healthz error. Once the
// gate is stable it can't be turned off, and the processor samples every trace: the change the
// warning describes.
func TestInvertMatchExcludesOnlyWhileTheGateIsOff(t *testing.T) {
	gate := "processor.tailsamplingprocessor.disableinvertdecisions"
	require.False(t, internalFeatureFlagDefaults[gate])
	t.Cleanup(func() { require.NoError(t, featuregate.GlobalRegistry().Set(gate, true)) })

	require.NoError(t, featuregate.GlobalRegistry().Set(gate, false))
	assert.ElementsMatch(t, []pcommon.TraceID{{2}}, sampledTraceIDs(t))

	require.NoError(t, featuregate.GlobalRegistry().Set(gate, true))
	assert.ElementsMatch(t, []pcommon.TraceID{{1}, {2}, {3}}, sampledTraceIDs(t))
}
