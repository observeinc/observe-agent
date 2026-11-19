package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

const testOtelConfig = `
receivers:
  otlp: {}
  file_log/a:
    include: [a.log]
  file_log/b:
    include: [b.log]
processors:
  batch: {}
  memory_limiter: {}
exporters:
  otlp_http/observe:
    endpoint: https://example.com
    sending_queue:
      enabled: true
  otlp_http/other:
    endpoint: https://other.example.com
  debug: {}
connectors:
  forward/logs: {}
  count: {}
extensions:
  health_check: {}
  file_storage: {}
service:
  extensions: [health_check, file_storage]
  pipelines:
    logs/in:
      receivers: [file_log/a, file_log/b]
      processors: [memory_limiter]
      exporters: [forward/logs, count]
    logs/out:
      receivers: [forward/logs]
      processors: [memory_limiter, batch]
      exporters: [otlp_http/observe]
    traces:
      receivers: otlp
      processors: [batch]
      exporters: otlp_http/observe, debug
`

func loadTestOtelConfig(t *testing.T) *OtelConfig {
	t.Helper()
	cfg, err := NewOtelConfigFromYAML([]byte(testOtelConfig))
	require.NoError(t, err)
	return cfg
}

func pipelineRefs(cfg *OtelConfig, pipeline, key string) []string {
	p, _ := cfg.pipelines()[pipeline].(map[string]any)
	return refList(p[key])
}

func TestOtelConfigList(t *testing.T) {
	cfg := loadTestOtelConfig(t)
	assert.Equal(t, []string{"logs/in", "logs/out", "traces"}, cfg.Pipelines())
	assert.Equal(t, []string{"file_log/a", "file_log/b", "otlp"}, cfg.Components(Receivers))
	assert.Equal(t, []string{"batch", "memory_limiter"}, cfg.Components(Processors))
	assert.Equal(t, []string{"debug", "otlp_http/observe", "otlp_http/other"}, cfg.Components(Exporters))
	assert.Equal(t, []string{"count", "forward/logs"}, cfg.Components(Connectors))
	assert.Equal(t, []string{"file_storage", "health_check"}, cfg.Components(Extensions))

	var ids []string
	for _, comp := range cfg.ComponentsOfType(Exporters, "otlp_http") {
		ids = append(ids, comp.ID)
	}
	assert.Equal(t, []string{"otlp_http/observe", "otlp_http/other"}, ids)
	assert.Nil(t, cfg.Component(Exporters, "missing"))
}

func TestComponentPathAccess(t *testing.T) {
	cfg := loadTestOtelConfig(t)
	for _, comp := range cfg.ComponentsOfType(Exporters, "otlp_http") {
		require.NoError(t, comp.Set(false, "sending_queue", "enabled"))
	}
	v, ok := cfg.Component(Exporters, "otlp_http/other").Get("sending_queue", "enabled")
	assert.True(t, ok)
	assert.Equal(t, false, v)

	comp := cfg.Component(Exporters, "otlp_http/observe")
	assert.Error(t, comp.Set(1, "endpoint", "nested"))
	assert.True(t, comp.Delete("endpoint"))
	assert.False(t, comp.Delete("endpoint"))
	_, ok = comp.Get("endpoint")
	assert.False(t, ok)

	// A component defined as null can still be written to.
	require.NoError(t, cfg.Component(Processors, "batch").Set(5, "timeout"))
	v, _ = cfg.Component(Processors, "batch").Get("timeout")
	assert.Equal(t, 5, v)
}

func TestOtelConfigRemoveComponent(t *testing.T) {
	cfg := loadTestOtelConfig(t)
	assert.True(t, cfg.RemoveComponent(Exporters, "debug"))
	assert.False(t, cfg.RemoveComponent(Exporters, "debug"))
	assert.Equal(t, []string{"otlp_http/observe"}, pipelineRefs(cfg, "traces", "exporters"))

	assert.True(t, cfg.RemoveComponent(Connectors, "forward/logs"))
	assert.Equal(t, []string{"count"}, pipelineRefs(cfg, "logs/in", "exporters"))
	assert.Empty(t, pipelineRefs(cfg, "logs/out", "receivers"))

	assert.True(t, cfg.RemoveComponent(Extensions, "file_storage"))
	assert.Equal(t, []string{"health_check"}, refList(cfg.service()["extensions"]))
}

func TestOtelConfigRemovePipelineAndUnused(t *testing.T) {
	cfg := loadTestOtelConfig(t)
	assert.True(t, cfg.RemovePipeline("logs/out"))
	assert.False(t, cfg.RemovePipeline("logs/out"))
	cfg.RemoveUnusedComponents()

	// Both connectors lost their receiving side, so they leave logs/in too.
	assert.Empty(t, cfg.Components(Connectors))
	assert.Empty(t, pipelineRefs(cfg, "logs/in", "exporters"))
	assert.Equal(t, []string{"debug", "otlp_http/observe"}, cfg.Components(Exporters))
	assert.Equal(t, []string{"file_storage", "health_check"}, cfg.Components(Extensions))
	assert.ErrorContains(t, cfg.Validate(), `pipeline "logs/in" has no exporters`)
}

func TestOtelConfigRetain(t *testing.T) {
	cfg := loadTestOtelConfig(t)
	assert.Error(t, cfg.RetainPipelines("missing"))
	require.NoError(t, cfg.RetainPipelines("traces"))
	require.NoError(t, cfg.RetainExtensions("health_check"))
	assert.Error(t, cfg.RetainExtensions("missing"))
	cfg.RemoveUnusedComponents()
	require.NoError(t, cfg.Validate())

	out, err := cfg.ToYAML()
	require.NoError(t, err)
	assert.Equal(t, `exporters:
  debug: {}
  otlp_http/observe:
    endpoint: https://example.com
    sending_queue:
      enabled: true
extensions:
  health_check: {}
processors:
  batch: {}
receivers:
  otlp: {}
service:
  extensions:
    - health_check
  pipelines:
    traces:
      exporters: otlp_http/observe, debug
      processors:
        - batch
      receivers: otlp
`, out)
}

func TestOtelConfigRenameComponent(t *testing.T) {
	cfg := loadTestOtelConfig(t)
	assert.Error(t, cfg.RenameComponent(Exporters, ParseComponentID("missing"), ParseComponentID("x")))

	// Rename only the type.
	from := ParseComponentID("otlp_http/observe")
	require.NoError(t, cfg.RenameComponent(Exporters, from, ComponentID{Type: "otlphttp", Name: from.Name}))
	assert.Equal(t, []string{"otlphttp/observe"}, pipelineRefs(cfg, "logs/out", "exporters"))
	assert.Equal(t, []string{"otlphttp/observe", "debug"}, pipelineRefs(cfg, "traces", "exporters"))

	// Rename a connector, which appears on both sides of pipelines.
	require.NoError(t, cfg.RenameComponent(Connectors, ParseComponentID("forward/logs"), ParseComponentID("forward")))
	assert.Equal(t, []string{"forward", "count"}, pipelineRefs(cfg, "logs/in", "exporters"))
	assert.Equal(t, []string{"forward"}, pipelineRefs(cfg, "logs/out", "receivers"))

	require.NoError(t, cfg.RenameComponent(Extensions, ParseComponentID("file_storage"), ParseComponentID("file_storage/agent")))
	assert.Equal(t, []string{"health_check", "file_storage/agent"}, refList(cfg.service()["extensions"]))
	require.NoError(t, cfg.Validate())
}

func TestOtelConfigRenameMerges(t *testing.T) {
	cfg := loadTestOtelConfig(t)
	require.NoError(t, cfg.RenameComponent(Exporters, ParseComponentID("otlp_http/other"), ParseComponentID("otlp_http/observe")))
	comp := cfg.Component(Exporters, "otlp_http/observe")
	endpoint, _ := comp.Get("endpoint")
	assert.Equal(t, "https://other.example.com", endpoint)
	enabled, _ := comp.Get("sending_queue", "enabled")
	assert.Equal(t, true, enabled)

	// A pipeline that lists both IDs keeps one.
	require.NoError(t, cfg.RenameComponent(Receivers, ParseComponentID("file_log/b"), ParseComponentID("file_log/a")))
	assert.Equal(t, []string{"file_log/a"}, pipelineRefs(cfg, "logs/in", "receivers"))
	include, _ := cfg.Component(Receivers, "file_log/a").Get("include")
	assert.Equal(t, []any{"b.log"}, include)
}

func TestOtelConfigValidate(t *testing.T) {
	cfg := loadTestOtelConfig(t)
	require.NoError(t, cfg.Validate())
	delete(cfg.raw["exporters"].(map[string]any), "debug")
	delete(cfg.raw["extensions"].(map[string]any), "file_storage")
	err := cfg.Validate()
	assert.ErrorContains(t, err, `pipeline "traces" references exporter "debug", which is not defined`)
	assert.ErrorContains(t, err, `service references extension "file_storage", which is not defined`)
}

func TestOtelConfigValidateReferenceTypes(t *testing.T) {
	for name, tc := range map[string]struct {
		pipeline string
		want     string
	}{
		"null item":      {"{receivers: [otlp], processors: [batch, null], exporters: [debug]}", `pipeline "logs" processors: item 1 is <nil>, not a component ID`},
		"number item":    {"{receivers: [otlp], exporters: [debug, 123]}", `pipeline "logs" exporters: item 1 is int, not a component ID`},
		"map value":      {"{receivers: {otlp: {}}, exporters: [debug]}", `pipeline "logs" receivers: is map[string]interface {}, not a list`},
		"extension item": {"{receivers: [otlp], exporters: [debug]}\n  extensions: [5]", `service extensions: item 0 is int, not a component ID`},
	} {
		t.Run(name, func(t *testing.T) {
			cfg, err := NewOtelConfigFromYAML([]byte("receivers: {otlp: {}}\nprocessors: {batch: {}}\nexporters: {debug: {}}\nservice:\n  pipelines:\n    logs: " + tc.pipeline + "\n"))
			require.NoError(t, err)
			err = cfg.Validate()
			assert.ErrorContains(t, err, tc.want)
			assert.NotContains(t, err.Error(), "has no", "a bad value is not reported as missing")
		})
	}
}

func TestOtelConfigConfmapRoundTrip(t *testing.T) {
	conf := confmap.NewFromStringMap(map[string]any{
		"exporters": map[string]any{"otlphttp/observe": map[string]any{"endpoint": "a"}},
		"service": map[string]any{"pipelines": map[string]any{
			"logs": map[string]any{"exporters": []any{"otlphttp/observe"}},
		}},
	})
	cfg := NewOtelConfigFromConfmap(conf)
	require.NoError(t, cfg.RenameComponent(Exporters, ParseComponentID("otlphttp/observe"), ParseComponentID("otlp_http/observe")))
	require.NoError(t, cfg.ApplyTo(conf))
	assert.Equal(t, "a", conf.Get("exporters::otlp_http/observe::endpoint"))
	assert.False(t, conf.IsSet("exporters::otlphttp/observe"))
	assert.Equal(t, []any{"otlp_http/observe"}, conf.Get("service::pipelines::logs::exporters"))
}

func TestTelemetryPathAccess(t *testing.T) {
	cfg, err := NewOtelConfigFromYAML([]byte(`
service:
  telemetry:
    metrics:
      readers:
        - pull: {exporter: {prometheus: {host: a}}}
        - pull: {exporter: {prometheus: {host: b}}}
`))
	require.NoError(t, err)
	telemetry := cfg.Telemetry()

	host, ok := telemetry.Get("metrics", "readers", "1", "pull", "exporter", "prometheus", "host")
	assert.True(t, ok)
	assert.Equal(t, "b", host)
	_, ok = telemetry.Get("metrics", "readers", "2")
	assert.False(t, ok)

	require.NoError(t, telemetry.Set(9999, "metrics", "readers", "0", "pull", "exporter", "prometheus", "port"))
	port, _ := telemetry.Get("metrics", "readers", "0", "pull", "exporter", "prometheus", "port")
	assert.Equal(t, 9999, port)
	assert.ErrorContains(t, telemetry.Set(1, "metrics", "readers", "5", "pull"), "out of range")
	assert.ErrorContains(t, telemetry.Set(1, "metrics", "readers", "0", "pull", "exporter", "prometheus", "host", "x"), "not a map or a list")

	// Deleting a list element shortens the list.
	assert.True(t, telemetry.Delete("metrics", "readers", "0"))
	readers, _ := telemetry.Get("metrics", "readers")
	assert.Len(t, readers, 1)
	assert.False(t, telemetry.Delete("metrics", "readers", "1"))

	// Set creates service.telemetry when it is missing.
	empty, err := NewOtelConfigFromYAML(nil)
	require.NoError(t, err)
	require.NoError(t, empty.Telemetry().Set("debug", "logs", "level"))
	out, err := empty.ToYAML()
	require.NoError(t, err)
	assert.Equal(t, "service:\n  telemetry:\n    logs:\n      level: debug\n", out)
}

// resolveWithEnv resolves a YAML config the way the collector does, so a list element written as
// ${env:...} holds an xconfmap.ExpandedValue instead of a string.
func resolveWithEnv(t *testing.T, yamlConfig string) *confmap.Conf {
	t.Helper()
	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs:              []string{"yaml:" + yamlConfig},
		ProviderFactories: []confmap.ProviderFactory{yamlprovider.NewFactory(), envprovider.NewFactory()},
	})
	require.NoError(t, err)
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	return conf
}

func TestOtelConfigKeepsEnvExpandedReferences(t *testing.T) {
	// The resolver keeps an ExpandedValue only when the env value parses as a non-string, so the
	// exporter needs an ID such as "true".
	t.Setenv("EXTRA_EXPORTER", "true")
	const yamlConfig = `
receivers:
  otlp: {}
exporters:
  otlphttp/observe: {}
  "true": {}
service:
  pipelines:
    traces:
      receivers: [otlp]
      exporters: [otlphttp/observe, "${env:EXTRA_EXPORTER}"]
`

	t.Run("rename keeps the other reference", func(t *testing.T) {
		conf := resolveWithEnv(t, yamlConfig)
		cfg := NewOtelConfigFromConfmap(conf)
		require.NoError(t, cfg.RenameComponent(Exporters, ParseComponentID("otlphttp/observe"), ParseComponentID("otlp_http/observe")))
		require.NoError(t, cfg.ApplyTo(conf))
		pipeline := xconfmap.ToStringMapRaw(conf)["service"].(map[string]any)["pipelines"].(map[string]any)["traces"].(map[string]any)
		assert.Equal(t, []any{"otlp_http/observe", xconfmap.ExpandedValue{Value: true, Original: "true"}}, pipeline["exporters"])
	})

	t.Run("remove unused keeps the referenced component", func(t *testing.T) {
		cfg := NewOtelConfigFromConfmap(resolveWithEnv(t, yamlConfig))
		cfg.RemoveUnusedComponents()
		assert.Equal(t, []string{"otlphttp/observe", "true"}, cfg.Components(Exporters))
	})

	t.Run("validate sees the reference", func(t *testing.T) {
		cfg := NewOtelConfigFromConfmap(resolveWithEnv(t, yamlConfig))
		require.True(t, cfg.RemoveComponent(Exporters, "otlphttp/observe"))
		delete(cfg.raw["exporters"].(map[string]any), "true")
		assert.ErrorContains(t, cfg.Validate(), `references exporter "true", which is not defined`)
	})
}

func TestOtelConfigKeepsEnvExpandedReferenceLists(t *testing.T) {
	// An env value that parses as a list makes the whole reference value an ExpandedValue.
	t.Setenv("EXPORTERS", "[otlphttp/observe, debug]")
	t.Setenv("EXTENSIONS", "[health_check]")
	const yamlConfig = `
receivers:
  otlp: {}
exporters:
  otlphttp/observe: {}
  debug: {}
extensions:
  health_check: {}
service:
  extensions: ${env:EXTENSIONS}
  pipelines:
    traces:
      receivers: [otlp]
      exporters: ${env:EXPORTERS}
`

	t.Run("validate accepts the list", func(t *testing.T) {
		cfg := NewOtelConfigFromConfmap(resolveWithEnv(t, yamlConfig))
		assert.NoError(t, cfg.Validate())
	})

	t.Run("remove unused keeps the referenced components", func(t *testing.T) {
		cfg := NewOtelConfigFromConfmap(resolveWithEnv(t, yamlConfig))
		cfg.RemoveUnusedComponents()
		assert.Equal(t, []string{"debug", "otlphttp/observe"}, cfg.Components(Exporters))
		assert.Equal(t, []string{"health_check"}, cfg.Components(Extensions))
	})

	t.Run("rename updates the list", func(t *testing.T) {
		conf := resolveWithEnv(t, yamlConfig)
		cfg := NewOtelConfigFromConfmap(conf)
		require.NoError(t, cfg.RenameComponent(Exporters, ParseComponentID("otlphttp/observe"), ParseComponentID("otlp_http/observe")))
		require.NoError(t, cfg.ApplyTo(conf))
		assert.Equal(t, []any{"otlp_http/observe", "debug"}, conf.Get("service::pipelines::traces::exporters"))
		require.NoError(t, cfg.Validate())
	})

	t.Run("an edit that changes nothing keeps the value", func(t *testing.T) {
		cfg := NewOtelConfigFromConfmap(resolveWithEnv(t, yamlConfig))
		cfg.editRefs(Exporters, func(id string) (string, bool) { return id, true })
		pipeline := cfg.pipelines()["traces"].(map[string]any)
		assert.IsType(t, xconfmap.ExpandedValue{}, pipeline["exporters"])
	})
}
