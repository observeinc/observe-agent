package observecol

import (
	"context"
	"os/exec"
	"strconv"
	"strings"
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
const healthzPolicy = `{name: healthz, type: string_attribute, string_attribute: {key: http.target, values: [/healthz]}}`
const dropDebugPolicy = `{name: drop-debug, type: drop, drop: {drop_sub_policy: [{name: debug, type: boolean_attribute, boolean_attribute: {key: debug, value: true}}]}}`

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
		})
	}
}

// Runs every entry's examples, under each of its component types.
func TestUpcomingChangeExamples(t *testing.T) {
	require.NotEmpty(t, upcomingChangeRules)
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
		{"one_of with equals", valid + "\n    match: {all: [{path: [a], equals: 1, one_of: [1, 2]}]}", "exactly one of"},
		{"one_of", valid + "\n    match: {all: [{path: [a], one_of: [1, 2]}]}", ""},
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
	changes := upcomingChanges(mapFromYAML(t, upcomingChangesConfig))
	require.Len(t, changes, 2)
	assert.Equal(t, "tail_sampling", changes[0].Component)
	assert.Equal(t, "tail_sampling/observe", changes[1].Component)
	assert.Contains(t, changes[0].Message, "no longer exclude traces")
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
