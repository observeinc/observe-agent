package observecol

import (
	"context"
	"testing"

	"github.com/observeinc/observe-agent/internal/connections"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"gopkg.in/yaml.v3"
)

// TestBuildResolverURIs pins the URI list buildResolverURIs assembles:
// fragments as `yaml:<body>` URIs, then --config values verbatim, then
// --set flags expanded into inline yaml URIs.
func TestBuildResolverURIs(t *testing.T) {
	frag := func(content string) connections.RenderedConfigFragment {
		return connections.RenderedConfigFragment{Name: "frag.yaml", Content: content}
	}

	cases := []struct {
		name         string
		fragments    []connections.RenderedConfigFragment
		otelConfigs  []string
		otelSets     []string
		want         []string
		wantErrExact string
	}{
		{
			name: "all inputs empty produces empty uri list",
			want: []string{},
		},
		{
			name: "fragments are inlined via yaml: scheme in order",
			fragments: []connections.RenderedConfigFragment{
				frag("receivers:\n  otlp:\n    protocols: {}\n"),
				frag("exporters:\n  otlp:\n    endpoint: a:4317\n"),
			},
			want: []string{
				"yaml:receivers:\n  otlp:\n    protocols: {}\n",
				"yaml:exporters:\n  otlp:\n    endpoint: a:4317\n",
			},
		},
		{
			name:        "otel config uris pass through verbatim",
			otelConfigs: []string{"file:/path/to/a", "env:OTEL_THING"},
			want:        []string{"file:/path/to/a", "env:OTEL_THING"},
		},
		{
			name:     "simple set becomes inline yaml key-value",
			otelSets: []string{"service=default"},
			want:     []string{"yaml:service: default"},
		},
		{
			name:     "dotted set keys are expanded with :: separator",
			otelSets: []string{"processors.batch.timeout=2s"},
			want:     []string{"yaml:processors::batch::timeout: 2s"},
		},
		{
			name:     "set trims whitespace around both key and value",
			otelSets: []string{"  processors.batch.timeout  =  2s  "},
			want:     []string{"yaml:processors::batch::timeout: 2s"},
		},
		{
			name:         "set without equals sign is rejected",
			otelSets:     []string{"processors.batch.timeout"},
			wantErrExact: "Value provided to --set flag is missing equal sign: processors.batch.timeout",
		},
		{
			name: "combined inputs preserve the fragments-configs-sets order",
			fragments: []connections.RenderedConfigFragment{
				frag("exporters:\n  otlp:\n    endpoint: a:4317\n"),
			},
			otelConfigs: []string{"file:/etc/otel.yaml"},
			otelSets:    []string{"service=test"},
			want: []string{
				"yaml:exporters:\n  otlp:\n    endpoint: a:4317\n",
				"file:/etc/otel.yaml",
				"yaml:service: test",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := buildResolverURIs(c.fragments, c.otelConfigs, c.otelSets)
			if c.wantErrExact != "" {
				require.Error(t, err)
				assert.EqualError(t, err, c.wantErrExact)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, c.want, got)
		})
	}
}

// TestBuildResolverURIs_FragmentURIsAreYamlProviderSafe feeds each fragment
// URI that buildResolverURIs emits through the real yamlprovider and asserts
// the decoded value matches the original Content, across multi-line YAML,
// leading/trailing whitespace, values containing colons, and a realistic
// yaml.Marshal output of a nested map.
func TestBuildResolverURIs_FragmentURIsAreYamlProviderSafe(t *testing.T) {
	marshal := func(t *testing.T, v map[string]any) string {
		t.Helper()
		b, err := yaml.Marshal(v)
		require.NoError(t, err)
		return string(b)
	}

	cases := []struct {
		name    string
		content string
		want    any
	}{
		{
			name:    "simple scalar map",
			content: "foo: bar\n",
			want:    map[string]any{"foo": "bar"},
		},
		{
			name:    "multi-line nested map",
			content: "exporters:\n  otlp:\n    endpoint: example.com:4317\n",
			want: map[string]any{
				"exporters": map[string]any{
					"otlp": map[string]any{"endpoint": "example.com:4317"},
				},
			},
		},
		{
			name:    "value with embedded colon",
			content: "endpoint: https://example.com:4317\n",
			want:    map[string]any{"endpoint": "https://example.com:4317"},
		},
		{
			name:    "leading and trailing whitespace is tolerated",
			content: "\n\n  foo: bar\n\n",
			want:    map[string]any{"foo": "bar"},
		},
		{
			name: "yaml.Marshal output of a realistic override map round-trips",
			content: marshal(t, map[string]any{
				"exporters": map[string]any{
					"otlp": map[string]any{
						"endpoint": "example.com:4317",
						"headers":  map[string]any{"X-Custom": "v"},
					},
				},
			}),
			want: map[string]any{
				"exporters": map[string]any{
					"otlp": map[string]any{
						"endpoint": "example.com:4317",
						"headers":  map[string]any{"X-Custom": "v"},
					},
				},
			},
		},
	}

	provider := yamlprovider.NewFactory().Create(confmap.ProviderSettings{})
	t.Cleanup(func() {
		_ = provider.Shutdown(context.Background())
	})

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			uris, err := buildResolverURIs(
				[]connections.RenderedConfigFragment{{Name: "frag.yaml", Content: c.content}},
				nil, nil,
			)
			require.NoError(t, err)
			require.Len(t, uris, 1)
			require.Equal(t, "yaml:"+c.content, uris[0],
				"fragment URI must be a plain concat of the yaml: scheme and Content")

			retrieved, err := provider.Retrieve(context.Background(), uris[0], nil)
			require.NoError(t, err, "yamlprovider must accept the fragment URI without error")
			raw, err := retrieved.AsRaw()
			require.NoError(t, err)
			assert.Equal(t, c.want, raw,
				"fragment content must round-trip through the yamlprovider unchanged")
		})
	}
}
