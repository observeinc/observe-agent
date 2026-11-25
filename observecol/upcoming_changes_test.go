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

// testRules keep the matcher tests independent of the entries in upcoming_changes.yaml.
const testRules = `
changes:
  - id: legacy-flag
    kind: receiver
    types: [example]
    upstream: v0.160.0
    remove_in: v2.30.0
    match: {all: [{path: [legacy], equals: true}]}
    message: legacy no longer works
    examples: {fires: [{legacy: true}], quiet: [{}]}
`

func useTestRules(t *testing.T) {
	saved := upcomingChangeRules
	upcomingChangeRules = mustLoadUpcomingChangeRules([]byte(testRules))
	t.Cleanup(func() { upcomingChangeRules = saved })
}

// Runs every entry's examples, under each of its component types.
func TestUpcomingChangeExamples(t *testing.T) {
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
	useTestRules(t)
	conf := mapFromYAML(t, `
receivers:
  example: {legacy: true}
  otlp: {legacy: true}
processors:
  example: {legacy: true}
`)
	changes := upcomingChanges(conf)
	require.Len(t, changes, 1)
	assert.Equal(t, UpcomingChange{ID: "legacy-flag", Component: "example", Message: "legacy no longer works"}, changes[0])
}

// Once an entry's remove_in release is tagged, the entry must be gone. Skipped without tags.
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
receivers:
  otlp: {}
  example/observe: {legacy: true}
  example: {legacy: true}
  example/clean: {}
processors:
  filter/legacy:
    legacy: true
service:
  pipelines:
    traces:
      receivers: [example]
`

func TestUpcomingChanges(t *testing.T) {
	useTestRules(t)
	changes := upcomingChanges(mapFromYAML(t, upcomingChangesConfig))
	require.Len(t, changes, 2)
	assert.Equal(t, "example", changes[0].Component)
	assert.Equal(t, "example/observe", changes[1].Component)
	assert.Equal(t, "legacy no longer works", changes[0].Message)
}

func TestUpcomingChangesConverterOnlyWarns(t *testing.T) {
	useTestRules(t)
	core, logs := observer.New(zapcore.WarnLevel)
	converter := newUpcomingChangesConverterFactory().Create(confmap.ConverterSettings{Logger: zap.New(core)})
	conf := confmap.NewFromStringMap(mapFromYAML(t, upcomingChangesConfig))
	before := conf.ToStringMap()

	require.NoError(t, converter.Convert(context.Background(), conf))

	assert.Equal(t, before, conf.ToStringMap())
	entries := logs.All()
	require.Len(t, entries, 2)
	assert.Equal(t, "example", entries[0].ContextMap()["component"])
	assert.Equal(t, "example/observe", entries[1].ContextMap()["component"])
}

func TestUpcomingChangesResolvesSettings(t *testing.T) {
	useTestRules(t)
	settings := generateCollectorSettings([]string{"yaml:" + upcomingChangesConfig})
	changes, err := UpcomingChanges(context.Background(), settings)
	require.NoError(t, err)
	require.Len(t, changes, 2)
}
