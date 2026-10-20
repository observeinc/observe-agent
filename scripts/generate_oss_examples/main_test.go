package main

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const testRendered = `
connectors:
  count: {}
  spanmetrics: {}
receivers:
  otlp: {}
  filestats/agent: {}
  nop: {}
processors:
  batch: {}
  filter/count: {}
  resource/agent_instance: {}
exporters:
  otlphttp/observe: {}
  nop: {}
extensions:
  file_storage: {}
  health_check: {}
service:
  extensions: [file_storage, health_check]
  pipelines:
    traces/spanmetrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [spanmetrics]
    metrics/spanmetrics:
      receivers: [spanmetrics]
      processors: [batch]
      exporters: [otlphttp/observe]
    logs/forward:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/observe, count]
    metrics/count-nop-in:
      receivers: [nop]
      exporters: [count]
    metrics/count-nop-out:
      receivers: [count]
      processors: [filter/count]
      exporters: [nop]
    metrics/agent-filestats:
      receivers: [filestats/agent]
      processors: [resource/agent_instance, batch]
      exporters: [otlphttp/observe]
`

func parse(t *testing.T, s string) map[string]any {
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(s), &cfg))
	return cfg
}

func Test_pruneKeepsOnlyWhatKeptPipelinesUse(t *testing.T) {
	cfg := parse(t, testRendered)
	require.NoError(t, prune(cfg, []string{"traces/spanmetrics", "metrics/spanmetrics", "logs/forward"}, []string{"health_check"}))

	assert.Equal(t, parse(t, `
connectors:
  spanmetrics: {}
receivers:
  otlp: {}
processors:
  batch: {}
exporters:
  otlphttp/observe: {}
extensions:
  health_check: {}
service:
  extensions: [health_check]
  pipelines:
    traces/spanmetrics:
      receivers: [otlp]
      processors: [batch]
      exporters: [spanmetrics]
    metrics/spanmetrics:
      receivers: [spanmetrics]
      processors: [batch]
      exporters: [otlphttp/observe]
    logs/forward:
      receivers: [otlp]
      processors: [batch]
      exporters: [otlphttp/observe]
`), cfg, "count is dropped from logs/forward because no kept pipeline receives from it")
}

func Test_pruneWithoutExtensions(t *testing.T) {
	cfg := parse(t, testRendered)
	require.NoError(t, prune(cfg, []string{"logs/forward"}, nil))
	assert.NotContains(t, cfg, "extensions")
	assert.NotContains(t, cfg["service"], "extensions")
	assert.NotContains(t, cfg, "connectors")
}

func Test_pruneErrors(t *testing.T) {
	for name, tc := range map[string]struct {
		rendered   string
		pipelines  []string
		extensions []string
	}{
		"missing pipeline":            {testRendered, []string{"traces/forward"}, nil},
		"missing extension":           {testRendered, []string{"logs/forward"}, []string{"pprof"}},
		"no service section":          {"receivers: {}", []string{"logs/forward"}, nil},
		"pipeline list is not a list": {"service: {pipelines: {logs/forward: {receivers: otlp, exporters: [x]}}}", []string{"logs/forward"}, nil},
		// metrics/spanmetrics alone receives from a connector that no kept pipeline exports to.
		"pipeline left without receivers": {testRendered, []string{"metrics/spanmetrics"}, nil},
	} {
		t.Run(name, func(t *testing.T) {
			assert.Error(t, prune(parse(t, tc.rendered), tc.pipelines, tc.extensions))
		})
	}
}

func Test_collectorVersionMatchesGoMod(t *testing.T) {
	goMod, err := os.ReadFile("../../go.mod")
	require.NoError(t, err)
	m := regexp.MustCompile(`(?m)^\s*go\.opentelemetry\.io/collector/otelcol v(\S+)`).FindSubmatch(goMod)
	require.NotNil(t, m)

	version, err := collectorVersion()
	require.NoError(t, err)
	assert.Equal(t, string(m[1]), version)
}

func Test_envVars(t *testing.T) {
	assert.Equal(t, []string{"A", "B_2"}, envVars([]byte("x: ${env:B_2}\ny: ${env:A}/v2 ${env:B_2}\nz: $notenv")))
}

func Test_checkNoCredentials(t *testing.T) {
	agentConfig := filepath.Join(t.TempDir(), "agent.yaml")
	require.NoError(t, os.WriteFile(agentConfig, []byte("observe_url: https://123.collect.example.com/\ntoken: ds:secret\n"), 0o600))

	assert.NoError(t, checkNoCredentials([]byte("endpoint: ${env:OBSERVE_COLLECTION_URL}/v2/otel"), agentConfig))
	assert.Error(t, checkNoCredentials([]byte("endpoint: https://123.collect.example.com/v2/otel"), agentConfig))
	assert.Error(t, checkNoCredentials([]byte("authorization: Bearer ds:secret"), agentConfig))
}

func Test_dropAgentInstanceID(t *testing.T) {
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(`
service:
  telemetry:
    resource:
      observe.agent.instance.id: abc
    metrics:
      readers:
        - pull:
            exporter:
              prometheus:
                host: localhost
                with_resource_constant_labels:
                  included:
                    - observe.agent.instance.id
`), &cfg))
	dropAgentInstanceID(cfg)
	telemetry := cfg["service"].(map[string]any)["telemetry"].(map[string]any)
	assert.NotContains(t, telemetry, "resource")
	prometheus := telemetry["metrics"].(map[string]any)["readers"].([]any)[0].(map[string]any)["pull"].(map[string]any)["exporter"].(map[string]any)["prometheus"].(map[string]any)
	assert.Equal(t, map[string]any{"host": "localhost"}, prometheus)
}

func Test_checkNoAgentFields(t *testing.T) {
	assert.NoError(t, checkNoAgentFields([]byte("host.name: ${env:HOST}")))
	assert.Error(t, checkNoAgentFields([]byte("observe.agent.instance.id: abc")))
	assert.Error(t, checkNoAgentFields([]byte("value: ${env:OBSERVE_AGENT_INSTANCE_ID}")))
}

func Test_writeAtomically(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "collector.yaml")
	require.NoError(t, writeAtomically(path, []byte("first")))
	require.NoError(t, writeAtomically(path, []byte("second")))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, "second", string(content))
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no temporary files are left behind")
}

func Test_writeAtomicallyFileMode(t *testing.T) {
	dir := t.TempDir()

	newFile := filepath.Join(dir, "new.yaml")
	require.NoError(t, writeAtomically(newFile, []byte("x")))
	info, err := os.Stat(newFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o644), info.Mode().Perm(), "a new example is readable by everyone")

	existing := filepath.Join(dir, "existing.yaml")
	require.NoError(t, os.WriteFile(existing, []byte("x"), 0o600))
	require.NoError(t, os.Chmod(existing, 0o640))
	require.NoError(t, writeAtomically(existing, []byte("y")))
	info, err = os.Stat(existing)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o640), info.Mode().Perm(), "replacing an example keeps its mode")
}

func Test_runRejectsBadOptions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "collector.yaml")
	require.NoError(t, os.WriteFile(path, []byte("existing"), 0o600))
	agentConfig := filepath.Join(dir, "agent.yaml")
	require.NoError(t, os.WriteFile(agentConfig, []byte("token: x\n"), 0o600))

	for name, opts := range map[string]options{
		"missing options":      {out: path},
		"invalid config mode":  {agentConfig: agentConfig, configMode: "solaris", pipelines: []string{"x"}, out: path},
		"missing agent config": {agentConfig: filepath.Join(dir, "missing.yaml"), configMode: "linux", pipelines: []string{"x"}, out: path},
		"missing otel config":  {agentConfig: agentConfig, otelConfig: filepath.Join(dir, "missing.yaml"), configMode: "linux", pipelines: []string{"x"}, out: path},
	} {
		t.Run(name, func(t *testing.T) {
			require.Error(t, run(opts))
			content, err := os.ReadFile(path)
			require.NoError(t, err)
			assert.Equal(t, "existing", string(content), "a failed run leaves the existing example untouched")
		})
	}
}
