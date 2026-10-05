package configconverter

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
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
// Without the converter the override lands on a separate component instead of
// merging.
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

// A user component that merely looks legacy must be left alone: the upstream
// type alias still resolves it, and only IDs in the table were broken by our
// renames.
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

// remapIDs identifies entries via a type assertion that yields "" on failure,
// so a malformed non-string entry must simply be carried through untouched.
func TestConvert_LeavesNonStringReferenceEntriesAlone(t *testing.T) {
	got := runConverter(t, map[string]string{"otlphttp/observe": "otlp_http/observe"}, map[string]any{
		"service": map[string]any{
			"pipelines": map[string]any{
				"logs": map[string]any{"exporters": []any{42, "otlphttp/observe", nil}},
			},
		},
	})
	pipeline := got["service"].(map[string]any)["pipelines"].(map[string]any)["logs"].(map[string]any)
	assert.Equal(t, []any{42, "otlp_http/observe", nil}, pipeline["exporters"])
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

// A remapped id must surface in the logs, since the converter rewrites it
// before the collector could emit its own deprecated alias warning. One entry
// per id, not one per rewritten leaf.
func TestConvert_WarnsOncePerRemappedID(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	conf := confmap.NewFromStringMap(map[string]any{
		"exporters": map[string]any{
			"otlphttp/observe": map[string]any{
				"compression":   "gzip",
				"sending_queue": map[string]any{"num_consumers": 20},
			},
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"logs": map[string]any{"exporters": []any{"otlphttp/observe"}},
			},
		},
	})
	conv := newFactory(map[string]string{"otlphttp/observe": "otlp_http/observe"}).
		Create(confmap.ConverterSettings{Logger: zap.New(core)})
	require.NoError(t, conv.Convert(context.Background(), conf))

	entries := logs.All()
	require.Len(t, entries, 1, "two rewritten leaves and a pipeline reference should still warn once")
	assert.Equal(t, "otlphttp/observe", entries[0].ContextMap()["deprecated_id"])
	assert.Equal(t, "otlp_http/observe", entries[0].ContextMap()["new_id"])
}

func TestConvert_DoesNotWarnWhenNothingRemapped(t *testing.T) {
	core, logs := observer.New(zap.WarnLevel)
	conf := confmap.NewFromStringMap(map[string]any{
		"exporters": map[string]any{"otlp_http/observe": map[string]any{"compression": "zstd"}},
	})
	conv := newFactory(map[string]string{"otlphttp/observe": "otlp_http/observe"}).
		Create(confmap.ConverterSettings{Logger: zap.New(core)})
	require.NoError(t, conv.Convert(context.Background(), conf))
	assert.Zero(t, logs.Len())
}

// NewFactory must apply the shipped table. The loop pins two invariants the
// rewrite depends on: a mapping never points at itself, and a canonical ID is
// never itself a legacy key, which would leave the result dependent on map
// iteration order.
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

func TestRemapDefinitionKey(t *testing.T) {
	c := &converter{mappings: map[string]string{
		"otlphttp/observe": "otlp_http/observe",
		"hostmetrics/host": "host_metrics/host",
		"debug":            "debug_exporter",
	}}
	cases := []struct {
		name, key, newKey, legacy string
		ok                        bool
	}{
		{"deeply nested remainder is preserved",
			"exporters::otlphttp/observe::sending_queue::num_consumers",
			"exporters::otlp_http/observe::sending_queue::num_consumers", "otlphttp/observe", true},
		{"bare id with no remainder",
			"exporters::debug", "exporters::debug_exporter", "debug", true},
		{"unqualified type in a different section",
			"receivers::hostmetrics/host::scrapers::cpu",
			"receivers::host_metrics/host::scrapers::cpu", "hostmetrics/host", true},
		{"id absent from the table", "exporters::otlphttp/mine", "", "", false},
		{"reference key is not a definition", "service::pipelines::logs::exporters", "", "", false},
		{"section on its own", "exporters", "", "", false},
		{"unknown top-level section", "service::telemetry::otlphttp/observe", "", "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			newKey, legacy, ok := c.remapDefinitionKey(tc.key)
			assert.Equal(t, tc.ok, ok)
			assert.Equal(t, tc.newKey, newKey)
			assert.Equal(t, tc.legacy, legacy)
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

func TestConvert_RewritesScalarAndCSVPipelineReferences(t *testing.T) {
	got := runConverter(t, map[string]string{"otlphttp/observe": "otlp_http/observe"}, map[string]any{
		"exporters": map[string]any{
			"otlphttp/observe": map[string]any{"endpoint": "https://example.collect.observeinc.com/v2/otel"},
		},
		"service": map[string]any{
			"pipelines": map[string]any{
				"logs":   map[string]any{"exporters": "otlphttp/observe"},
				"traces": map[string]any{"exporters": "otlphttp/observe, count"},
			},
		},
	})

	pipelines := got["service"].(map[string]any)["pipelines"].(map[string]any)
	assert.Equal(t, "otlp_http/observe", pipelines["logs"].(map[string]any)["exporters"])
	assert.Equal(t, []any{"otlp_http/observe", "count"}, pipelines["traces"].(map[string]any)["exporters"])
}

func TestConvert_PreservesExpandedValueMetadata(t *testing.T) {
	expanded := xconfmap.ExpandedValue{Value: 123456, Original: "123456"}
	conf := confmap.NewFromStringMap(map[string]any{
		"exporters": map[string]any{
			"otlphttp/observe": map[string]any{
				"headers": map[string]any{"x-tenant-id": expanded},
			},
		},
	})
	conv := newFactory(map[string]string{"otlphttp/observe": "otlp_http/observe"}).
		Create(confmap.ConverterSettings{})
	require.NoError(t, conv.Convert(context.Background(), conf))

	raw := xconfmap.ToStringMapRaw(conf)
	headers := raw["exporters"].(map[string]any)["otlp_http/observe"].(map[string]any)["headers"].(map[string]any)
	got, ok := headers["x-tenant-id"].(xconfmap.ExpandedValue)
	require.True(t, ok, "expected ExpandedValue after remap, got %T", headers["x-tenant-id"])
	assert.Equal(t, expanded, got)
}
