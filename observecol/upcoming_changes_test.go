package observecol

import (
	"bytes"
	"context"
	"os/exec"
	"slices"
	"strconv"
	"strings"
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

// useAgentFeatureGates sets the feature gates the agent applies at startup for the test's duration.
func useAgentFeatureGates(t *testing.T) {
	ctx := context.Background()
	resetFeatureGates(t, ctx)
	t.Cleanup(func() { resetFeatureGates(t, ctx) })
}

const errorsPolicy = `{name: errors, type: status_code, status_code: {status_codes: [ERROR]}}`
const healthzInverted = `{name: no-healthz, type: string_attribute, string_attribute: {key: http.target, values: [/healthz], invert_match: true}}`
const healthzPolicy = `{name: healthz, type: string_attribute, string_attribute: {key: http.target, values: [/healthz]}}`
const dropDebugPolicy = `{name: drop-debug, type: drop, drop: {drop_sub_policy: [{name: debug, type: boolean_attribute, boolean_attribute: {key: debug, value: true}}]}}`

// Each case runs through the bundled tail_sampling processor with disableinvertdecisions off, as
// the agent sets it, and on, as it is once the gate is stable. The check must fire exactly when the
// sampled traces differ.
func TestReliesOnInvertMatch(t *testing.T) {
	tests := []struct {
		name     string
		policies string
		want     bool
	}{
		{"inverted next to another policy", `[` + errorsPolicy + `, ` + healthzInverted + `]`, true},
		{"two inverted policies", `[` + healthzInverted + `, {name: slow, type: numeric_attribute, numeric_attribute: {key: duration, min_value: 1, max_value: 2, invert_match: true}}]`, true},
		{"inverted next to a drop policy", `[` + healthzInverted + `, ` + dropDebugPolicy + `]`, true},
		{"lone not wrapping an inverted policy", `[{name: not, type: not, not: {not_sub_policy: ` + healthzInverted + `}}]`, true},
		{"not wrapping an inverted policy next to another policy", `[` + errorsPolicy + `, {name: not, type: not, not: {not_sub_policy: ` + healthzInverted + `}}]`, true},
		{"lone inverted policy", `[` + healthzInverted + `]`, false},
		{"not wrapping a plain policy", `[` + errorsPolicy + `, {name: not, type: not, not: {not_sub_policy: ` + healthzPolicy + `}}]`, false},
		{"inverted inside and", `[` + errorsPolicy + `, {name: and, type: and, and: {and_sub_policy: [` + healthzInverted + `]}}]`, false},
		{"inverted inside drop", `[` + errorsPolicy + `, {name: drop, type: drop, drop: {drop_sub_policy: [` + healthzInverted + `]}}]`, false},
		{"invert_match false", `[` + errorsPolicy + `, ` + healthzPolicy + `]`, false},
		{"no invert_match", `[` + errorsPolicy + `]`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, reliesOnInvertMatch(mapFromYAML(t, "policies: "+tt.policies)))

			useAgentFeatureGates(t)
			before := sampledTraceIDs(t, tt.policies)
			require.NoError(t, featuregate.GlobalRegistry().Set(disableInvertDecisionsGate, true))
			after := sampledTraceIDs(t, tt.policies)
			assert.Equal(t, tt.want, !assert.ObjectsAreEqual(before, after), "sampled with the gate off %v, on %v", before, after)
		})
	}
}

// Users who already enabled disableinvertdecisions have the new behavior, so nothing changes for them.
func TestInvertMatchCheckSkipsWhenTheGateIsEnabled(t *testing.T) {
	cfg := mapFromYAML(t, "policies: ["+errorsPolicy+", "+healthzInverted+"]")
	check := upcomingChangeChecks["tail_sampling_invert_match"]
	useAgentFeatureGates(t)
	assert.True(t, check(cfg))
	require.NoError(t, featuregate.GlobalRegistry().Set(disableInvertDecisionsGate, true))
	assert.False(t, check(cfg))
}

// Runs every entry's examples, under each of its component types.
func TestUpcomingChangeExamples(t *testing.T) {
	require.NotEmpty(t, upcomingChangeRules)
	useAgentFeatureGates(t)
	for _, r := range upcomingChangeRules {
		for _, typ := range r.Types {
			id := typ + "/example"
			for i, cfg := range r.Examples.Fires {
				t.Run(r.ID+"/"+typ+"/fires/"+strconv.Itoa(i), func(t *testing.T) {
					changes := upcomingChanges(map[string]any{r.Kind + "s": map[string]any{id: cfg}})
					assert.Contains(t, changes, UpcomingChange{ID: r.ID, Component: id, Message: strings.TrimSpace(r.Message)})
				})
			}
			for i, cfg := range r.Examples.Quiet {
				t.Run(r.ID+"/"+typ+"/quiet/"+strconv.Itoa(i), func(t *testing.T) {
					for _, change := range upcomingChanges(map[string]any{r.Kind + "s": map[string]any{id: cfg}}) {
						assert.NotEqual(t, r.ID, change.ID)
					}
				})
			}
		}
	}
}

func TestUpcomingChangeRulesValidation(t *testing.T) {
	valid := `
  - id: example
    kind: receiver
    types: [example]
    upstream: v0.160.0
    remove_in: v2.30.0
    message: something changes
    examples: {fires: [{}]}`
	tests := []struct {
		name    string
		content string
		err     string
	}{
		{"valid", valid, ""},
		{"duplicate id", valid + valid, "defined twice"},
		{"unknown kind", strings.Replace(valid, "kind: receiver", "kind: pipeline", 1), "kind must be one of"},
		{"bad version", strings.Replace(valid, "remove_in: v2.30.0", "remove_in: 2.30", 1), "must look like"},
		{"no fires example", strings.Replace(valid, "examples: {fires: [{}]}", "examples: {quiet: [{}]}", 1), "examples.fires"},
		{"unknown check", valid + "\n    match: {check: nope}", "unknown check"},
		{"two operators", valid + "\n    match: {all: [{path: [a], equals: 1, absent: true}]}", "exactly one of"},
		{"no path", valid + "\n    match: {all: [{equals: 1}]}", "exactly one of"},
		{"misspelled match key", valid + "\n    match: {al: [{path: [a], equals: 1}]}", "field al not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadUpcomingChangeRules([]byte("changes:" + tt.content))
			if tt.err == "" {
				assert.NoError(t, err)
			} else {
				assert.ErrorContains(t, err, tt.err)
			}
		})
	}
}

// An entry only matches components of its own kind and types.
func TestUpcomingChangesOnlyMatchTheirTypes(t *testing.T) {
	conf := mapFromYAML(t, `
receivers:
  otlp: {}
  prometheus: {}
exporters:
  prometheusremotewrite:
    add_metric_suffixes: false
processors:
  tail_sampling:
    policies: [`+errorsPolicy+`, `+healthzInverted+`]
`)
	useAgentFeatureGates(t)
	changes := upcomingChanges(conf)
	require.Len(t, changes, 1)
	assert.Equal(t, "tail-sampling-invert-match", changes[0].ID)
}

// remove_in is Phase 3: once that release is tagged, the entry must be gone. Skipped without tags.
func TestUpcomingChangesNotExpired(t *testing.T) {
	out, err := exec.Command("git", "describe", "--tags", "--abbrev=0", "--match", "v[0-9]*").Output()
	if err != nil {
		t.Skip("no release tag available")
	}
	latest := strings.TrimSpace(string(out))
	for _, r := range upcomingChangeRules {
		assert.Negative(t, compareVersions(latest, r.RemoveIn), "%s should have been removed in %s; the latest release is %s", r.ID, r.RemoveIn, latest)
	}
}

func compareVersions(a, b string) int {
	pa, pb := strings.Split(strings.TrimPrefix(a, "v"), "."), strings.Split(strings.TrimPrefix(b, "v"), ".")
	for i := 0; i < 3; i++ {
		x, _ := strconv.Atoi(strings.SplitN(pa[min(i, len(pa)-1)], "-", 2)[0])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x - y
		}
	}
	return 0
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
	useAgentFeatureGates(t)
	changes := upcomingChanges(mapFromYAML(t, upcomingChangesConfig))
	require.Len(t, changes, 2)
	assert.Equal(t, "tail_sampling", changes[0].Component)
	assert.Equal(t, "tail_sampling/observe", changes[1].Component)
	assert.Contains(t, changes[0].Message, "no longer exclude traces")
}

func TestUpcomingChangesConverterOnlyWarns(t *testing.T) {
	useAgentFeatureGates(t)
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
	useAgentFeatureGates(t)
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

// sampledTraceIDs runs a /healthz error (ID 1), an /api error (ID 2) and an /api OK trace (ID 3)
// through a tail_sampling processor with the given policies, and returns the IDs it samples.
func sampledTraceIDs(t *testing.T, policies string) []pcommon.TraceID {
	factory := tailsamplingprocessor.NewFactory()
	cfg := factory.CreateDefaultConfig()
	require.NoError(t, confmap.NewFromStringMap(mapFromYAML(t, "policies: "+policies)).Unmarshal(cfg))

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
	slices.SortFunc(sampled, func(a, b pcommon.TraceID) int { return bytes.Compare(a[:], b[:]) })
	return sampled
}

// The agent keeps disableinvertdecisions off, so invert_match excludes the /healthz error. Once the
// gate is stable it can't be turned off, and the processor samples every trace: the change the
// warning describes.
func TestInvertMatchExcludesOnlyWhileTheGateIsOff(t *testing.T) {
	require.False(t, internalFeatureFlagDefaults[disableInvertDecisionsGate])
	policies := "[" + errorsPolicy + ", " + healthzInverted + "]"
	useAgentFeatureGates(t)
	assert.Equal(t, []pcommon.TraceID{{2}}, sampledTraceIDs(t, policies))

	require.NoError(t, featuregate.GlobalRegistry().Set(disableInvertDecisionsGate, true))
	assert.Equal(t, []pcommon.TraceID{{1}, {2}, {3}}, sampledTraceIDs(t, policies))
}
