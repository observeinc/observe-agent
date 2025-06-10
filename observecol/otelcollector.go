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
				DefaultScheme: "file",
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

func GetOtelCollectorSettings(ctx context.Context) (*otelcol.CollectorSettings, func(), error) {
	observeAgentConfigs, cleanup, err := connections.SetupAndGetConfigFiles(ctx)
	if err != nil {
		return nil, cleanup, err
	}
	URIs := append(observeAgentConfigs, otelConfigs...)
	// This loop is copied directly from the otelcol `set` flag handling.
	for _, s := range otelSets {
		idx := strings.Index(s, "=")
		if idx == -1 {
			return nil, cleanup, fmt.Errorf("Value provided to --set flag is missing equal sign: %s", s)
		}
		URIs = append(URIs, "yaml:"+strings.TrimSpace(strings.ReplaceAll(s[:idx], ".", "::"))+": "+strings.TrimSpace(s[idx+1:]))
	}
	return generateCollectorSettings(URIs), cleanup, nil
}

func GetOtelCollector(ctx context.Context) (*otelcol.Collector, func(), error) {
	settings, cleanup, err := GetOtelCollectorSettings(ctx)
	if err != nil {
		return nil, cleanup, err
	}

	col, err := otelcol.NewCollector(*settings)
	if err != nil {
		return nil, cleanup, err
	}
	return col, cleanup, nil
}
