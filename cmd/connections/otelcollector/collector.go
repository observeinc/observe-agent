package observeotel

import (
	"net/url"
	"observe/agent/build"
	"os"

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
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

func GenerateCollectorSettings() *collector.CollectorSettings {
	otelConfigPath := viper.GetString("otel_config")
	providerSet := confmap.ProviderSettings{}
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
				URIs: []string{otelConfigPath},
				Providers: makeMapProvidersMap(
					fileprovider.NewWithSettings(providerSet),
					envprovider.NewWithSettings(providerSet),
					yamlprovider.NewWithSettings(providerSet),
					httpprovider.NewWithSettings(providerSet),
					httpsprovider.NewWithSettings(providerSet),
				),
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

func SetEnvVars() error {
	collector_url, token := viper.GetString("observe_url"), viper.GetString("token")
	endpoint, err := url.JoinPath(collector_url, "/v2/otel")
	if err != nil {
		return err
	}
	fsPath := viper.GetString("filestorage_path")
	// Setting values from the Observe agent config as env vars to fill in the OTEL collector config
	os.Setenv("OBSERVE_ENDPOINT", endpoint)
	os.Setenv("OBSERVE_TOKEN", "Bearer "+token)
	if fsPath != "" {
		os.Setenv("FILESTORAGE_PATH", fsPath)
	}
	return nil
}

func GetOtelCollectorCommand() *cobra.Command {
	otelconfig := GenerateCollectorSettings()
	cmd := otelcol.NewCommand(*otelconfig)
	return cmd
}
