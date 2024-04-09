package observeotel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

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

func generateCollectorSettings(otelConfig string) *collector.CollectorSettings {
	providerSet := confmap.ProviderSettings{}
	set := &collector.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: baseFactories,
		ConfigProviderSettings: collector.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{otelConfig},
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

func StartCollector(wg *sync.WaitGroup) error {
	wg.Add(1)
	ctx := context.Background()
	endpoint, token := viper.GetString("observe_url"), viper.GetString("token")
	otelConfig := viper.GetString("otel_config")
	fsPath := viper.GetString("filestorage_path")
	// Setting values from the Observe agent config as env vars to fill in the OTEL collector config
	os.Setenv("OBSERVE_ENDPOINT", endpoint)
	os.Setenv("OBSERVE_TOKEN", "Bearer "+token)
	if fsPath != "" {
		os.Setenv("FILESTORAGE_PATH", fsPath)
	}
	if otelConfig == "" {
		otelConfig = filepath.Join("packaging/macos/", "otel-collector.yaml")
	}
	fmt.Fprintln(os.Stderr, "Using OTEL config file:", otelConfig)
	set := generateCollectorSettings(otelConfig)
	col, err := collector.NewCollector(*set)
	if err != nil {
		fmt.Fprintf(os.Stderr, "unable to start agent: %v\n", err)
		os.Exit(1)
	}

	colErrorChannel := make(chan error, 1)
	// col.Run blocks until receiving a SIGTERM signal, so needs to be started
	// asynchronously, but it will exit early if an error occurs on startup
	go func() {
		colErrorChannel <- col.Run(ctx)
	}()

	// wait for an error to returned
	return <-colErrorChannel
}
