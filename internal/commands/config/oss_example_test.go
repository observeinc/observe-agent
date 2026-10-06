package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

const ossExampleOTLPURL = "http://127.0.0.1:24518"

// Test_OSSHostExample runs examples/oss-collector/host/collector.yaml with its exporters
// redirected to local files, and checks the deployment environment on the telemetry it forwards.
// The example itself is generated from the agent configuration by `make generate-oss-examples`,
// which the code-gen-check workflow keeps up to date.
func Test_OSSHostExample(t *testing.T) {
	outDir := t.TempDir()
	examplePath := filepath.Join(getCurPath(), "../../../examples/oss-collector/host/collector.yaml")
	configPath := filepath.Join(t.TempDir(), "collector.yaml")
	writeTestableOSSExample(t, examplePath, configPath, outDir)

	t.Setenv("DEPLOYMENT_ENVIRONMENT", "prod")
	setupConfig(t, snapshotTest{
		agentConfigPath: "test/snap2-empty-agent-config.yaml",
		otelConfigPath:  configPath,
		packageType:     Linux,
	})
	t.Setenv("FILESTORAGE_PATH", t.TempDir())
	startCollector(t)

	postOTLP(t, ossExampleOTLPURL, "/v1/traces", buildTraces(t))
	postOTLP(t, ossExampleOTLPURL, "/v1/logs", buildLogs(t))
	postOTLP(t, ossExampleOTLPURL, "/v1/metrics", buildMetrics(t))

	want := map[string]envAttrs{
		"no-env":     {Name: "prod", Legacy: "prod"},
		"legacy-env": {Name: "staging", Legacy: "staging"},
		"new-env":    {Name: "dev", Legacy: "dev"},
	}
	t.Run("forwarded traces", func(t *testing.T) {
		assert.Equal(t, want, waitForResources(t, filepath.Join(outDir, "traces.jsonl"), tracesResources))
	})
	t.Run("forwarded logs", func(t *testing.T) {
		assert.Equal(t, want, waitForResources(t, filepath.Join(outDir, "logs.jsonl"), logsResources))
	})
	t.Run("forwarded metrics", func(t *testing.T) {
		assert.Equal(t, want, waitForResources(t, filepath.Join(outDir, "metrics.jsonl"), metricsResources))
	})
	t.Run("RED metrics", func(t *testing.T) {
		assert.Equal(t, want, waitForResources(t, filepath.Join(outDir, "red_metrics.jsonl"), metricsResources))
	})
}

// writeTestableOSSExample copies the example with a local OTLP receiver, file exporters, and only
// the env cloud detector, and without the host metrics pipeline.
func writeTestableOSSExample(t *testing.T, examplePath, configPath, outDir string) {
	raw, err := os.ReadFile(examplePath)
	require.NoError(t, err)
	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &cfg))

	cfg["receivers"].(map[string]any)["otlp"] = map[string]any{
		"protocols": map[string]any{"http": map[string]any{"endpoint": "127.0.0.1:24518"}},
	}
	// Cloud metadata endpoints are not reachable (or differ) on CI machines.
	cfg["processors"].(map[string]any)["resourcedetection/cloud"].(map[string]any)["detectors"] = []any{"env"}
	for _, id := range []string{"spanmetrics", "spanmetrics/summary"} {
		cfg["connectors"].(map[string]any)[id].(map[string]any)["metrics_flush_interval"] = "100ms"
	}
	fileExporter := func(name string) map[string]any {
		return map[string]any{"path": filepath.Join(outDir, name), "flush_interval": "100ms"}
	}
	cfg["exporters"] = map[string]any{
		"file/traces":      fileExporter("traces.jsonl"),
		"file/logs":        fileExporter("logs.jsonl"),
		"file/metrics":     fileExporter("metrics.jsonl"),
		"file/red_metrics": fileExporter("red_metrics.jsonl"),
	}
	service := cfg["service"].(map[string]any)
	service["telemetry"] = map[string]any{"metrics": map[string]any{"level": "none"}}
	pipelines := service["pipelines"].(map[string]any)
	delete(pipelines, "metrics/host_monitoring_host")
	exportersByPipeline := map[string]string{
		"traces/forward":              "file/traces",
		"logs/forward":                "file/logs",
		"metrics/forward":             "file/metrics",
		"metrics/spanmetrics":         "file/red_metrics",
		"metrics/spanmetrics/summary": "file/red_metrics",
	}
	for name, exporter := range exportersByPipeline {
		require.Contains(t, pipelines, name, "example pipeline %q", name)
		pipelines[name].(map[string]any)["exporters"] = []any{exporter}
	}

	out, err := yaml.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(configPath, out, 0o600))
}
