package configconverter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
)

func runConverter(t *testing.T, mappings map[string]string, in map[string]any) map[string]any {
	t.Helper()
	conf := confmap.NewFromStringMap(in)
	conv := newFactory(mappings).Create(confmap.ConverterSettings{})
	require.NoError(t, conv.Convert(context.Background(), conf))
	return conf.ToStringMap()
}

// A legacy block and an existing canonical block must combine as a deep merge,
// with the user's legacy leaves winning and untouched bundled leaves surviving.
// This is the case that breaks today: the override lands on a separate component
// instead of merging.
func TestConvert_MergesLegacyBlockIntoCanonical(t *testing.T) {
	got := runConverter(t, map[string]string{"otlphttp/observe": "otlp_http/observe"}, map[string]any{
		"exporters": map[string]any{
			"otlp_http/observe": map[string]any{
				"endpoint":    "https://example.collect.observeinc.com/v2/otel",
				"compression": "zstd",
				"sending_queue": map[string]any{
					"num_consumers": 4,
					"queue_size":    100,
				},
			},
			"otlphttp/observe": map[string]any{
				"compression": "gzip",
				"sending_queue": map[string]any{
					"num_consumers": 20,
				},
			},
		},
	})

	exporters := got["exporters"].(map[string]any)
	assert.NotContains(t, exporters, "otlphttp/observe")
	require.Contains(t, exporters, "otlp_http/observe")

	observe := exporters["otlp_http/observe"].(map[string]any)
	assert.Equal(t, "https://example.collect.observeinc.com/v2/otel", observe["endpoint"],
		"bundled leaf the user did not override should survive")
	assert.Equal(t, "gzip", observe["compression"],
		"user value should win on a conflicting leaf")

	queue := observe["sending_queue"].(map[string]any)
	assert.Equal(t, 20, queue["num_consumers"], "user value should win on a nested conflicting leaf")
	assert.Equal(t, 100, queue["queue_size"], "sibling bundled leaf should survive the merge")
}

// When nothing already occupies the canonical ID the block is simply renamed.
func TestConvert_RenamesWhenNoCanonicalCounterpart(t *testing.T) {
	got := runConverter(t, map[string]string{"hostmetrics/custom": "host_metrics/custom"}, map[string]any{
		"receivers": map[string]any{
			"hostmetrics/custom": map[string]any{
				"collection_interval": "60s",
			},
		},
	})

	receivers := got["receivers"].(map[string]any)
	assert.NotContains(t, receivers, "hostmetrics/custom")
	assert.Equal(t, map[string]any{"collection_interval": "60s"}, receivers["host_metrics/custom"])
}

func TestConvert_RewritesPipelineAndExtensionReferences(t *testing.T) {
	got := runConverter(t, map[string]string{
		"filelog/host_monitoring": "file_log/host_monitoring",
		"otlphttp/observe":        "otlp_http/observe",
		"filestorage":             "file_storage",
	}, map[string]any{
		"receivers": map[string]any{
			"filelog/host_monitoring": map[string]any{"include": []any{"/var/log/syslog"}},
		},
		"exporters": map[string]any{
			"otlphttp/observe": map[string]any{"endpoint": "https://example.collect.observeinc.com/v2/otel"},
		},
		"extensions": map[string]any{
			"filestorage": map[string]any{"directory": "/var/lib/observe-agent/filestorage"},
		},
		"service": map[string]any{
			"extensions": []any{"filestorage", "health_check"},
			"pipelines": map[string]any{
				"logs/host-monitoring": map[string]any{
					"receivers": []any{"filelog/host_monitoring"},
					"exporters": []any{"otlphttp/observe", "count"},
				},
			},
		},
	})

	service := got["service"].(map[string]any)
	assert.Equal(t, []any{"file_storage", "health_check"}, service["extensions"])

	pipeline := service["pipelines"].(map[string]any)["logs/host-monitoring"].(map[string]any)
	assert.Equal(t, []any{"file_log/host_monitoring"}, pipeline["receivers"])
	assert.Equal(t, []any{"otlp_http/observe", "count"}, pipeline["exporters"],
		"unmapped entries in the list should keep their position and value")
}

// A user component that merely looks legacy is none of our business: the
// upstream type alias resolves it, and only IDs in the table were broken by us.
func TestConvert_LeavesUnmappedComponentsAlone(t *testing.T) {
	in := map[string]any{
		"receivers": map[string]any{
			"filelog/mycustom": map[string]any{"include": []any{"/var/log/app.log"}},
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"logs": map[string]any{"receivers": []any{"filelog/mycustom"}},
			},
		},
	}
	got := runConverter(t, map[string]string{"filelog/host_monitoring": "file_log/host_monitoring"}, in)
	assert.Equal(t, in, got)
}

func TestConvert_EmptyTableIsNoOp(t *testing.T) {
	in := map[string]any{
		"exporters": map[string]any{
			"otlphttp/observe": map[string]any{"endpoint": "https://example.collect.observeinc.com/v2/otel"},
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"logs": map[string]any{"exporters": []any{"otlphttp/observe"}},
			},
		},
	}
	assert.Equal(t, in, runConverter(t, map[string]string{}, in))
}

// The shipped table is applied through NewFactory; exercise that path so the
// wiring is covered even while the table is empty.
func TestNewFactory_AppliesShippedTable(t *testing.T) {
	conf := confmap.NewFromStringMap(map[string]any{
		"exporters": map[string]any{"otlp_http/observe": map[string]any{"compression": "zstd"}},
	})
	conv := NewFactory().Create(confmap.ConverterSettings{})
	require.NoError(t, conv.Convert(context.Background(), conf))

	for legacy, canonical := range LegacyComponentIDs {
		assert.NotEqual(t, legacy, canonical, "a mapping to itself would loop pointlessly")
		assert.NotContains(t, LegacyComponentIDs, canonical,
			"canonical id %q must not itself be a legacy key, which would make the result order-dependent", canonical)
	}
}

func TestSplitComponentKey(t *testing.T) {
	cases := []struct {
		key               string
		section, id, rest string
		ok                bool
	}{
		{"exporters::otlphttp/observe::sending_queue::num_consumers", "exporters", "otlphttp/observe", "sending_queue::num_consumers", true},
		{"exporters::debug", "exporters", "debug", "", true},
		{"receivers::hostmetrics/host::scrapers::cpu", "receivers", "hostmetrics/host", "scrapers::cpu", true},
		{"service::pipelines::logs::exporters", "", "", "", false},
		{"exporters", "", "", "", false},
	}
	for _, c := range cases {
		t.Run(c.key, func(t *testing.T) {
			section, id, rest, ok := splitComponentKey(c.key)
			assert.Equal(t, c.ok, ok)
			assert.Equal(t, c.section, section)
			assert.Equal(t, c.id, id)
			assert.Equal(t, c.rest, rest)
		})
	}
}

func TestIsReferenceKey(t *testing.T) {
	assert.True(t, isReferenceKey("service::extensions"))
	assert.True(t, isReferenceKey("service::pipelines::logs::receivers"))
	assert.True(t, isReferenceKey("service::pipelines::metrics/host-monitoring::exporters"))
	assert.True(t, isReferenceKey("service::pipelines::traces::processors"))
	assert.False(t, isReferenceKey("service::telemetry::resource"))
	assert.False(t, isReferenceKey("extensions::file_storage::directory"))
	assert.False(t, isReferenceKey("service::pipelines::logs::unknown"))
}
