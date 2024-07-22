package observeotel

import (
	"observe-agent/build"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"

	"go.opentelemetry.io/collector/otelcol"
	collector "go.opentelemetry.io/collector/otelcol"
)

func makeMapProvidersMap(providers ...confmap.Provider) map[string]confmap.Provider {
	ret := make(map[string]confmap.Provider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}

func GenerateCollectorSettings(URIs []string) *collector.CollectorSettings {
	buildInfo := component.BuildInfo{
		Command:     "observe-agent",
		Description: "Observe Distribution of Opentelemetry Collector",
		Version:     build.Version,
	}
	set := &collector.CollectorSettings{
		BuildInfo: buildInfo,
		Factories: components,
		ConfigProviderSettings: collector.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: URIs,
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

func GetOtelCollectorCommand(otelconfig *collector.CollectorSettings) *cobra.Command {
	cmd := otelcol.NewCommand(*otelconfig)
	return cmd
}
