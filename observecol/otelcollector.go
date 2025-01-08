package observecol

import (
	"github.com/observeinc/observe-agent/build"

	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/otelcol"
)

func GenerateCollectorSettings() *otelcol.CollectorSettings {
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

func GenerateCollectorSettingsWithConfigFiles(configFiles []string) *otelcol.CollectorSettings {
	set := GenerateCollectorSettings()
	set.ConfigProviderSettings.ResolverSettings.URIs = configFiles
	return set
}

func GetOtelCollectorCommand(otelconfig *otelcol.CollectorSettings) *cobra.Command {
	cmd := otelcol.NewCommand(*otelconfig)
	return cmd
}
