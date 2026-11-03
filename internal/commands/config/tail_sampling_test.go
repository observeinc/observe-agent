package config

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor/processortest"
	"gopkg.in/yaml.v3"
)

const (
	sampleErrors        = `{name: sample-errors, type: status_code, status_code: {status_codes: [ERROR]}}`
	excludeHealthChecks = `{name: exclude-health-checks, type: string_attribute, string_attribute: {key: http.target, values: [/healthz], invert_match: true}}`
	dropDebug           = `{name: drop-debug, type: drop, drop: {drop_sub_policy: [{name: debug, type: boolean_attribute, boolean_attribute: {key: debug, value: true}}]}}`
)

type tailSamplingCase struct {
	name string
	// processors is the otel_config_overrides processors section; pipeline lists them in order.
	processors string
	pipeline   []string
	// processor is the tail_sampling processor under test, and runsAs its ID after the agent's
	// invert_match swap.
	processor string
	runsAs    string
	// unsupported is set when the processor relies on invert_match excluding traces but
	// tailsampling_legacy can't run its configuration; it appears in the swap error.
	unsupported string
	// sampled, when set, is what the processor samples from a /healthz error (1), an /api error (2)
	// and an /api OK trace (3).
	sampled []pcommon.TraceID
}

var tailSamplingCases = []tailSamplingCase{
	// Relies on invert_match excluding traces: runs as tailsampling_legacy.
	{
		name:       "string_attribute inverted next to status_code",
		processors: "tail_sampling:\n  policies: [" + sampleErrors + ", " + excludeHealthChecks + "]",
		pipeline:   []string{"tail_sampling"},
		processor:  "tail_sampling", runsAs: "tailsampling_legacy",
		sampled: []pcommon.TraceID{{2}},
	},
	{
		name: "numeric_attribute inverted next to probabilistic",
		processors: "tail_sampling/test:\n  policies:\n" +
			"    - {name: sample-ten-percent, type: probabilistic, probabilistic: {sampling_percentage: 10}}\n" +
			"    - {name: exclude-small, type: numeric_attribute, numeric_attribute: {key: http.response.body.size, min_value: 0, max_value: 100, invert_match: true}}",
		pipeline:  []string{"tail_sampling/test"},
		processor: "tail_sampling/test", runsAs: "tailsampling_legacy/test",
	},
	{
		name: "boolean_attribute inverted next to latency",
		processors: "tail_sampling/test:\n  policies:\n" +
			"    - {name: sample-slow, type: latency, latency: {threshold_ms: 5000}}\n" +
			"    - {name: exclude-synthetic, type: boolean_attribute, boolean_attribute: {key: synthetic, value: true, invert_match: true}}",
		pipeline:  []string{"tail_sampling/test"},
		processor: "tail_sampling/test", runsAs: "tailsampling_legacy/test",
	},
	{
		name: "two inverted policies",
		processors: "tail_sampling/test:\n  policies:\n    - " + excludeHealthChecks + "\n" +
			"    - {name: exclude-small, type: numeric_attribute, numeric_attribute: {key: http.response.body.size, min_value: 0, max_value: 100, invert_match: true}}",
		pipeline:  []string{"tail_sampling/test"},
		processor: "tail_sampling/test", runsAs: "tailsampling_legacy/test",
	},
	{
		name:       "inverted next to a drop policy and status_code",
		processors: "tail_sampling/test:\n  policies: [" + dropDebug + ", " + sampleErrors + ", " + excludeHealthChecks + "]",
		pipeline:   []string{"tail_sampling/test"},
		processor:  "tail_sampling/test", runsAs: "tailsampling_legacy/test",
		sampled: []pcommon.TraceID{{2}},
	},
	{
		name:       "pipeline order and other processors are kept",
		processors: "batch: {}\ntail_sampling/test:\n  policies: [" + sampleErrors + ", " + excludeHealthChecks + "]",
		pipeline:   []string{"batch", "tail_sampling/test"},
		processor:  "tail_sampling/test", runsAs: "tailsampling_legacy/test",
	},

	// Same decisions on both versions: stays on the upstream tail_sampling processor.
	{
		name:       "no invert_match",
		processors: "tail_sampling/test:\n  policies: [" + sampleErrors + ", {name: sample-slow, type: latency, latency: {threshold_ms: 5000}}]",
		pipeline:   []string{"tail_sampling/test"},
		processor:  "tail_sampling/test", runsAs: "tail_sampling/test",
		sampled: []pcommon.TraceID{{1}, {2}},
	},
	{
		name:       "lone inverted policy",
		processors: "tail_sampling/test:\n  policies: [" + excludeHealthChecks + "]",
		pipeline:   []string{"tail_sampling/test"},
		processor:  "tail_sampling/test", runsAs: "tail_sampling/test",
		sampled: []pcommon.TraceID{{2}, {3}},
	},
	{
		name:       "lone inverted policy and a drop policy",
		processors: "tail_sampling/test:\n  policies: [" + dropDebug + ", " + excludeHealthChecks + "]",
		pipeline:   []string{"tail_sampling/test"},
		processor:  "tail_sampling/test", runsAs: "tail_sampling/test",
		sampled: []pcommon.TraceID{{2}, {3}},
	},
	{
		name: "inverted inside and",
		processors: "tail_sampling/test:\n  policies:\n    - " + sampleErrors + "\n" +
			"    - {name: errors-except-health-checks, type: and, and: {and_sub_policy: [" + sampleErrors + ", " + excludeHealthChecks + "]}}",
		pipeline:  []string{"tail_sampling/test"},
		processor: "tail_sampling/test", runsAs: "tail_sampling/test",
		sampled: []pcommon.TraceID{{1}, {2}},
	},
	{
		name: "inverted inside composite",
		processors: "tail_sampling/test:\n  policies:\n    - " + sampleErrors + "\n" +
			"    - {name: composite, type: composite, composite: {max_total_spans_per_second: 1000, policy_order: [exclude-health-checks], " +
			"composite_sub_policy: [" + excludeHealthChecks + "], rate_allocation: [{policy: exclude-health-checks, percent: 100}]}}",
		pipeline:  []string{"tail_sampling/test"},
		processor: "tail_sampling/test", runsAs: "tail_sampling/test",
	},
	{
		name: "inverted inside a drop policy",
		processors: "tail_sampling/test:\n  policies:\n    - " + sampleErrors + "\n" +
			"    - {name: drop-non-api, type: drop, drop: {drop_sub_policy: [{name: not-api, type: string_attribute, string_attribute: {key: http.target, values: [/api], invert_match: true}}]}}",
		pipeline:  []string{"tail_sampling/test"},
		processor: "tail_sampling/test", runsAs: "tail_sampling/test",
		sampled: []pcommon.TraceID{{2}},
	},
	{
		name: "invert_match false",
		processors: "tail_sampling/test:\n  policies:\n    - " + sampleErrors + "\n" +
			"    - {name: health-checks, type: string_attribute, string_attribute: {key: http.target, values: [/healthz], invert_match: false}}",
		pipeline:  []string{"tail_sampling/test"},
		processor: "tail_sampling/test", runsAs: "tail_sampling/test",
		sampled: []pcommon.TraceID{{1}, {2}},
	},

	// Relies on invert_match excluding traces, but tailsampling_legacy can't run the configuration:
	// stays on tail_sampling, where the inverted policy no longer excludes traces.
	{
		name:       "num_shards, added after v0.151.0",
		processors: "tail_sampling/test:\n  num_shards: 2\n  policies: [" + sampleErrors + ", " + excludeHealthChecks + "]",
		pipeline:   []string{"tail_sampling/test"},
		processor:  "tail_sampling/test", runsAs: "tail_sampling/test", unsupported: "num_shards",
		sampled: []pcommon.TraceID{{1}, {2}, {3}},
	},
	{
		name: "rate_limiting burst_capacity, added after v0.151.0",
		processors: "tail_sampling/test:\n  policies:\n" +
			"    - {name: rate-limit, type: rate_limiting, rate_limiting: {spans_per_second: 100, burst_capacity: 200}}\n    - " + excludeHealthChecks,
		pipeline:  []string{"tail_sampling/test"},
		processor: "tail_sampling/test", runsAs: "tail_sampling/test", unsupported: "burst_capacity",
	},
	{
		name: "tailsampling_legacy ID already defined",
		processors: "tail_sampling/test:\n  policies: [" + sampleErrors + ", " + excludeHealthChecks + "]\n" +
			"tailsampling_legacy/test:\n  policies: [" + sampleErrors + "]",
		pipeline:  []string{"tail_sampling/test", "tailsampling_legacy/test"},
		processor: "tail_sampling/test", runsAs: "tail_sampling/test", unsupported: "tailsampling_legacy/test is already defined",
	},
}

func writeTailSamplingAgentConfig(t *testing.T, tc tailSamplingCase) string {
	var processors map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(tc.processors), &processors))
	agentConfig := map[string]any{
		"observe_url":          "https://test.collect.observeinc.com/",
		"token":                "12345678901234567890:abcdefghijklmnopqrstuvwxyzABCDEF",
		"omit_base_components": true,
		"forwarding":           map[string]any{"enabled": false},
		"health_check":         map[string]any{"enabled": false},
		"internal_telemetry":   map[string]any{"enabled": false},
		"host_monitoring":      map[string]any{"enabled": false},
		"self_monitoring":      map[string]any{"enabled": false},
		"otel_config_overrides": map[string]any{
			"receivers":  map[string]any{"nop": map[string]any{}},
			"exporters":  map[string]any{"nop": map[string]any{}},
			"processors": processors,
			"service": map[string]any{"pipelines": map[string]any{
				"traces": map[string]any{"receivers": []string{"nop"}, "processors": tc.pipeline, "exporters": []string{"nop"}},
			}},
		},
	}
	out, err := yaml.Marshal(agentConfig)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "observe-agent.yaml")
	require.NoError(t, os.WriteFile(path, out, 0o600))
	return path
}

func resolveCollectorConf(t *testing.T, ctx context.Context, settings *otelcol.CollectorSettings) *confmap.Conf {
	resolver, err := confmap.NewResolver(settings.ConfigProviderSettings.ResolverSettings)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resolver.Shutdown(ctx)) })
	conf, err := resolver.Resolve(ctx)
	require.NoError(t, err)
	return conf
}

func newHealthCheckTrace(id byte, target string, status ptrace.StatusCode) ptrace.Traces {
	td := ptrace.NewTraces()
	span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID{id})
	span.SetSpanID(pcommon.SpanID{id})
	span.Attributes().PutStr("http.target", target)
	span.Status().SetCode(status)
	return td
}

// Runs each case's agent configuration through the agent's config path, with its invert_match swap,
// and checks which processor each tail_sampling processor runs on, that every processor builds and
// starts with the agent's factories, and, where given, what it samples.
func Test_TailSamplingInvertMatch(t *testing.T) {
	for _, tc := range tailSamplingCases {
		t.Run(tc.name, func(t *testing.T) {
			setupConfig(t, snapshotTest{agentConfigPath: writeTailSamplingAgentConfig(t, tc), packageType: Linux})
			ctx := logger.WithCtx(context.Background(), logger.GetNop())
			require.NoError(t, observecol.ApplyFeatureGates(ctx))
			settings, err := observecol.GetOtelCollectorSettings(ctx)
			require.NoError(t, err)

			swaps, err := observecol.InvertMatchSwaps(ctx, settings)
			require.NoError(t, err)
			switch {
			case tc.unsupported != "":
				require.Len(t, swaps, 1)
				assert.Equal(t, tc.processor, swaps[0].Processor)
				assert.ErrorContains(t, swaps[0].Err, tc.unsupported)
			case tc.runsAs != tc.processor:
				require.Len(t, swaps, 1)
				assert.Equal(t, tc.processor, swaps[0].Processor)
				assert.Equal(t, tc.runsAs, swaps[0].LegacyID)
				assert.NoError(t, swaps[0].Err)
			default:
				assert.Empty(t, swaps)
			}

			conf := resolveCollectorConf(t, ctx, settings)
			wantPipeline := slices.Clone(tc.pipeline)
			for i, id := range wantPipeline {
				if id == tc.processor {
					wantPipeline[i] = tc.runsAs
				}
			}
			var gotPipeline []string
			for _, id := range conf.Get("service::pipelines::traces::processors").([]any) {
				gotPipeline = append(gotPipeline, id.(string))
			}
			assert.Equal(t, wantPipeline, gotPipeline)

			factories, err := settings.Factories()
			require.NoError(t, err)
			for _, idStr := range wantPipeline {
				var id component.ID
				require.NoError(t, id.UnmarshalText([]byte(idStr)))
				factory, ok := factories.Processors[id.Type()]
				require.True(t, ok, "the agent's components don't include %s", id.Type())
				sub, err := conf.Sub("processors::" + idStr)
				require.NoError(t, err)
				cfg := factory.CreateDefaultConfig()
				require.NoError(t, sub.Unmarshal(cfg))
				require.NoError(t, confmap.Validate(cfg))

				set := processortest.NewNopSettings(id.Type())
				set.ID = id
				sink := new(consumertest.TracesSink)
				proc, err := factory.CreateTraces(ctx, set, cfg, sink)
				require.NoError(t, err)
				require.NoError(t, proc.Start(ctx, componenttest.NewNopHost()))
				if idStr == tc.runsAs && tc.sampled != nil {
					require.NoError(t, proc.ConsumeTraces(ctx, newHealthCheckTrace(1, "/healthz", ptrace.StatusCodeError)))
					require.NoError(t, proc.ConsumeTraces(ctx, newHealthCheckTrace(2, "/api", ptrace.StatusCodeError)))
					require.NoError(t, proc.ConsumeTraces(ctx, newHealthCheckTrace(3, "/api", ptrace.StatusCodeOk)))
				}
				require.NoError(t, proc.Shutdown(ctx))
				if idStr == tc.runsAs && tc.sampled != nil {
					var sampled []pcommon.TraceID
					for _, td := range sink.AllTraces() {
						sampled = append(sampled, td.ResourceSpans().At(0).ScopeSpans().At(0).Spans().At(0).TraceID())
					}
					assert.ElementsMatch(t, tc.sampled, sampled)
				}
			}
		})
	}
}
