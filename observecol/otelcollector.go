package observecol

import (
	"context"
	"fmt"
	"strings"

	"github.com/observeinc/observe-agent/build"
	"github.com/observeinc/observe-agent/internal/connections"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"
)

func generateCollectorSettings(URIs []string) *otelcol.CollectorSettings {
	buildInfo := component.BuildInfo{
		Command:     "observe-agent",
		Description: "Observe Distribution of Opentelemetry Collector",
		Version:     build.Version,
	}
	set := &otelcol.CollectorSettings{
		BuildInfo: buildInfo,
		Factories: components,
		ConfigProviderSettings: otelcol.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:          URIs,
				DefaultScheme: "env",
				ProviderFactories: []confmap.ProviderFactory{
					fileprovider.NewFactory(),
					envprovider.NewFactory(),
					yamlprovider.NewFactory(),
					httpprovider.NewFactory(),
					httpsprovider.NewFactory(),
				},
			},
		},
	}
	return set
}

// buildResolverURIs assembles the heterogeneous URI list handed to the
// otelcol confmap resolver. It appends, in order:
//
//  1. Each in-memory fragment as an inline `yaml:<body>` URI. Fragments are
//     RenderedConfigFragment structs (no longer file paths), so every entry
//     must be transformed before it can live alongside the string-typed
//     otelConfigs / otelSets URIs. The yamlprovider's `yaml:` scheme is a
//     marker, not an RFC 3986 URI: its Retrieve implementation strips the
//     prefix and hands the remainder directly to yaml.Unmarshal, so no
//     escaping or trimming is needed -- fragment Content is produced by our
//     templates or by yaml.Marshal and is always well-formed YAML, and YAML
//     is whitespace-tolerant.
//  2. The user's `--config` flag values verbatim. Those are already URIs
//     for the file/http/env providers.
//  3. Each `--set key=value` flag expanded into an inline `yaml:` URI,
//     using the same parsing as the upstream otelcol `set` flag.
//
// The result slice is pre-sized to the full final length so it does not
// re-grow across the three append steps.
func buildResolverURIs(fragments []connections.RenderedConfigFragment, otelConfigs, otelSets []string) ([]string, error) {
	URIs := make([]string, 0, len(fragments)+len(otelConfigs)+len(otelSets))
	for _, f := range fragments {
		URIs = append(URIs, "yaml:"+f.Content)
	}
	URIs = append(URIs, otelConfigs...)
	for _, s := range otelSets {
		idx := strings.Index(s, "=")
		if idx == -1 {
			return nil, fmt.Errorf("Value provided to --set flag is missing equal sign: %s", s)
		}
		URIs = append(URIs, "yaml:"+strings.TrimSpace(strings.ReplaceAll(s[:idx], ".", "::"))+": "+strings.TrimSpace(s[idx+1:]))
	}
	return URIs, nil
}

func GetOtelCollectorSettings(ctx context.Context) (*otelcol.CollectorSettings, error) {
	fragments, err := connections.SetupAndGetConfigs(ctx)
	if err != nil {
		return nil, err
	}
	URIs, err := buildResolverURIs(fragments, otelConfigs, otelSets)
	if err != nil {
		return nil, err
	}
	return generateCollectorSettings(URIs), nil
}

func GetOtelCollector(ctx context.Context) (*otelcol.Collector, error) {
	settings, err := GetOtelCollectorSettings(ctx)
	if err != nil {
		return nil, err
	}
	col, err := otelcol.NewCollector(*settings)
	if err != nil {
		return nil, err
	}
	return col, nil
}
