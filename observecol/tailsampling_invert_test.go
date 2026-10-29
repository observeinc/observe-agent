package observecol

import (
	"context"
	"testing"

	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/tailsamplingprocessor"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
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

// The patched tail_sampling processor keeps the pre-v0.144.0 invert_match semantics: a matching
// inverted policy excludes the trace even when another policy samples it, and a non-matching one
// only samples when no other policy returns NotSampled.
func TestTailSamplingKeepsInvertMatchDecisions(t *testing.T) {
	factory := tailsamplingprocessor.NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.NewFromStringMap(map[string]any{
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
	}).Unmarshal(cfg))

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
	require.ElementsMatch(t, []pcommon.TraceID{{2}}, sampled)
}
