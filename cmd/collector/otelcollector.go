package observeotel

import (
	"observe/agent/build"

	"github.com/open-telemetry/opentelemetry-collector-contrib/connector/countconnector"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/storage/filestorage"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/processor/transformprocessor"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filelogreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/filestatsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/hostmetricsreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/journaldreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/prometheusreceiver"
	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/windowseventlogreceiver"
	"github.com/spf13/cobra"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/exporter/loggingexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/processor/batchprocessor"
	"go.opentelemetry.io/collector/processor/memorylimiterprocessor"

	"go.opentelemetry.io/collector/otelcol"
	collector "go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
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
		Factories: baseFactories,
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

// Each module's factories needs to be manually included here for the parser to then handle that config.
func baseFactories() (otelcol.Factories, error) {
	var factories otelcol.Factories
	var err error

	if factories.Extensions, err = extension.MakeFactoryMap(
		healthcheckextension.NewFactory(),
		filestorage.NewFactory(),
	); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Receivers, err = receiver.MakeFactoryMap(
		otlpreceiver.NewFactory(),
		hostmetricsreceiver.NewFactory(),
		filestatsreceiver.NewFactory(),
		filelogreceiver.NewFactory(),
		prometheusreceiver.NewFactory(),
		journaldreceiver.NewFactory(),
		windowseventlogreceiver.NewFactory(),
	); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Exporters, err = exporter.MakeFactoryMap(
		loggingexporter.NewFactory(),
		debugexporter.NewFactory(),
		otlphttpexporter.NewFactory(),
	); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Processors, err = processor.MakeFactoryMap(
		transformprocessor.NewFactory(),
		memorylimiterprocessor.NewFactory(),
		batchprocessor.NewFactory(),
		resourcedetectionprocessor.NewFactory(),
	); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Connectors, err = connector.MakeFactoryMap(countconnector.NewFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	return factories, err
}

func GetOtelCollectorCommand(otelconfig *collector.CollectorSettings) *cobra.Command {
	cmd := otelcol.NewCommand(*otelconfig)
	return cmd
}
