package observeotel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/open-telemetry/opentelemetry-collector-contrib/extension/healthcheckextension"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/debugexporter"
	"go.opentelemetry.io/collector/exporter/otlphttpexporter"
	"go.opentelemetry.io/collector/extension"

	"go.opentelemetry.io/collector/otelcol"
	collector "go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
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

func generateCollectorSettings() *collector.CollectorSettings {
	providerSet := confmap.ProviderSettings{}
	set := &collector.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: baseFactories,
		ConfigProviderSettings: collector.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs: []string{filepath.Join("conf.d", "otel-collector.yaml")},
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

	if factories.Extensions, err = extension.MakeFactoryMap(healthcheckextension.NewFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Receivers, err = receiver.MakeFactoryMap(otlpreceiver.NewFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Exporters, err = exporter.MakeFactoryMap(debugexporter.NewFactory(), otlphttpexporter.NewFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Processors, err = processor.MakeFactoryMap(processortest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Connectors, err = connector.MakeFactoryMap(connectortest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	return factories, err
}

func StartCollector(wg *sync.WaitGroup) error {
	wg.Add(1)
	ctx := context.Background()
	endpoint, token := viper.GetString("observe_url"), viper.GetString("token")
	// Setting values from the Observe agent config as env vars to fill in the OTEL collector config
	os.Setenv("OBSERVE_ENDPOINT", endpoint)
	os.Setenv("OBSERVE_TOKEN", "Bearer "+token)
	set := generateCollectorSettings()
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
