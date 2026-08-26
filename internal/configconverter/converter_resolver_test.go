package configconverter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func resolveURIs(t *testing.T, uris []string, log *zap.Logger) *confmap.Conf {
	t.Helper()
	if log == nil {
		log = zap.NewNop()
	}
	resolver, err := confmap.NewResolver(confmap.ResolverSettings{
		URIs: uris,
		ProviderFactories: []confmap.ProviderFactory{
			yamlprovider.NewFactory(),
			envprovider.NewFactory(),
		},
		DefaultScheme:      "env",
		ConverterFactories: []confmap.ConverterFactory{NewFactory()},
		ConverterSettings:  confmap.ConverterSettings{Logger: log},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = resolver.Shutdown(context.Background()) })
	conf, err := resolver.Resolve(context.Background())
	require.NoError(t, err)
	return conf
}

func exporterCompression(t *testing.T, conf *confmap.Conf) string {
	t.Helper()
	exporters, ok := conf.ToStringMap()["exporters"].(map[string]any)
	require.True(t, ok)
	observe, ok := exporters["otlp_http/observe"].(map[string]any)
	require.True(t, ok, "canonical exporter id should be present: %v", exporters)
	assert.NotContains(t, exporters, "otlphttp/observe")
	compression, ok := observe["compression"].(string)
	require.True(t, ok)
	return compression
}

// Later URIs (--config, --set, otel_config_overrides) must beat earlier bundled
// values when both already use the canonical ID. This is the production shape
// after bundled templates were renamed.
func TestResolver_LaterCanonicalOverrideWins(t *testing.T) {
	conf := resolveURIs(t, []string{
		"yaml:exporters:\n  otlp_http/observe:\n    compression: zstd\n",
		"yaml:exporters:\n  otlp_http/observe:\n    compression: gzip\n",
	}, nil)
	assert.Equal(t, "gzip", exporterCompression(t, conf))
}

// A user still naming the legacy ID (typical otel_config_overrides from an
// older install) must merge on top of bundled canonical config.
func TestResolver_UserLegacyOverrideWinsOverBundledCanonical(t *testing.T) {
	conf := resolveURIs(t, []string{
		"yaml:exporters:\n  otlp_http/observe:\n    compression: zstd\n    endpoint: https://example\n",
		"yaml:exporters:\n  otlphttp/observe:\n    compression: gzip\n",
	}, nil)
	assert.Equal(t, "gzip", exporterCompression(t, conf))
	exporters := conf.ToStringMap()["exporters"].(map[string]any)
	assert.Equal(t, "https://example", exporters["otlp_http/observe"].(map[string]any)["endpoint"])
}

func TestResolver_StockCanonicalConfigDoesNotWarn(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	_ = resolveURIs(t, []string{
		"yaml:exporters:\n  otlp_http/observe:\n    compression: zstd\n",
	}, zap.New(core))
	assert.Zero(t, logs.Len(), "bundled canonical ids must not emit deprecation warnings")
}

func TestResolver_PreservesExpandedEnvMetadata(t *testing.T) {
	t.Setenv("TENANT_ID", "123456")
	conf := resolveURIs(t, []string{
		"yaml:exporters:\n  otlphttp/observe:\n    headers:\n      x-tenant-id: ${env:TENANT_ID}\n",
	}, nil)
	raw := xconfmap.ToStringMapRaw(conf)
	headers := raw["exporters"].(map[string]any)["otlp_http/observe"].(map[string]any)["headers"].(map[string]any)
	got, ok := headers["x-tenant-id"].(xconfmap.ExpandedValue)
	require.True(t, ok, "expected ExpandedValue, got %T (%v)", headers["x-tenant-id"], headers["x-tenant-id"])
	assert.Equal(t, 123456, got.Value)
	assert.Equal(t, "123456", got.Original)
}
