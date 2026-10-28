package observecol

import (
	"context"
	"testing"

	tailsamplinglegacyprocessor "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/featuregate"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
)

func newTrace(id byte, target string, status ptrace.StatusCode) ptrace.Traces {
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID{id})
	span.SetSpanID(pcommon.SpanID{id})
	span.Attributes().PutStr("http.target", target)
	span.Status().SetCode(status)
	return td
}

// healthCheckPolicies samples errors and uses invert_match to exclude /healthz.
var healthCheckPolicies = map[string]any{
	"policies": []any{
		map[string]any{
			"name":        "sample-errors",
			"type":        "status_code",
			"status_code": map[string]any{"status_codes": []any{"ERROR"}},
		},
		map[string]any{
			"name": "exclude-health-checks",
			"type": "string_attribute",
			"string_attribute": map[string]any{
				"key":          "http.target",
				"values":       []any{"/healthz"},
				"invert_match": true,
			},
		},
	},
}

// sampledTraceIDs runs a /healthz error, an /api error and an /api OK trace through the processor
// and returns the IDs it samples.
func sampledTraceIDs(t *testing.T, factory processor.Factory) []pcommon.TraceID {
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.NewFromStringMap(healthCheckPolicies).Unmarshal(cfg))

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

func TestTailSamplingLegacyKeepsInvertMatchDecisions(t *testing.T) {
	gate := "processor.tailsamplinglegacyprocessor.disableinvertdecisions"
	require.NoError(t, featuregate.GlobalRegistry().Set(gate, internalFeatureFlagDefaults[gate]))
	t.Cleanup(func() { require.NoError(t, featuregate.GlobalRegistry().Set(gate, true)) })

	require.ElementsMatch(t, []pcommon.TraceID{{2}}, sampledTraceIDs(t, tailsamplinglegacyprocessor.NewFactory()))
}

func TestTailSamplingNoLongerExcludesWithInvertMatch(t *testing.T) {
	require.ElementsMatch(t, []pcommon.TraceID{{1}, {2}, {3}}, sampledTraceIDs(t, tailsamplingprocessor.NewFactory()))
}
