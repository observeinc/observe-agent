/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/spf13/cobra"
	component "go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/connector/connectortest"
	"go.opentelemetry.io/collector/exporter"
	"go.opentelemetry.io/collector/exporter/exportertest"
	"go.opentelemetry.io/collector/extension"
	"go.opentelemetry.io/collector/extension/extensiontest"
	"go.opentelemetry.io/collector/otelcol"
	collector "go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processortest"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// startCmd represents the start command
var startCmd = &cobra.Command{
	Use:   "start",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		var wg sync.WaitGroup
		err := startCollector(wg)
		if err != nil {
			fmt.Errorf("error: %e", err)
		}
		wg.Wait()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// func generateConfig() *service.Config {
// 	return &service.Config{
// 		Telemetry: telemetry.Config{
// 			Logs: telemetry.LogsConfig{
// 				Level:             zapcore.DebugLevel,
// 				Development:       true,
// 				Encoding:          "console",
// 				DisableCaller:     true,
// 				DisableStacktrace: true,
// 				OutputPaths:       []string{"stderr", "./output-logs"},
// 				ErrorOutputPaths:  []string{"stderr", "./error-output-logs"},
// 				InitialFields:     map[string]any{"fieldKey": "filed-value"},
// 			},
// 			Metrics: telemetry.MetricsConfig{
// 				Level:   configtelemetry.LevelNormal,
// 				Address: ":8080",
// 			},
// 		},
// 		// Extensions: extensions.Config{component.MustNewID("nop")},
// 		// Pipelines: pipelines.Config{
// 		// 	component.MustNewID("traces"): {
// 		// 		Receivers:  []component.ID{component.MustNewID("nop")},
// 		// 		Processors: []component.ID{component.MustNewID("nop")},
// 		// 		Exporters:  []component.ID{component.MustNewID("nop")},
// 		// 	},
// 		// },
// 	}
// }

func startCollector(wg sync.WaitGroup) error {
	ctx := context.Background()
	set := collector.CollectorSettings{
		BuildInfo: component.NewDefaultBuildInfo(),
		Factories: NopFactories,
		ConfigProviderSettings: collector.ConfigProviderSettings{
			ResolverSettings: confmap.ResolverSettings{
				URIs:      []string{filepath.Join("conf.d", "otel-collector.yaml")},
				Providers: makeMapProvidersMap(newFailureProvider()),
			},
		},
	}
	col, err := collector.NewCollector(set)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("We created our collector", col)
	fmt.Println(col.GetState())
	if err != nil {
		fmt.Errorf("err: %e", err)
	}
	colErrorChannel := make(chan error, 1)

	// col.Run blocks until receiving a SIGTERM signal, so needs to be started
	// asynchronously, but it will exit early if an error occurs on startup
	go func() {
		colErrorChannel <- col.Run(ctx)
	}()

	// wait until the collector server is in the Running state
	go func() {
		for {
			state := col.GetState()
			fmt.Println("current state", state)
			if state == collector.StateRunning {
				colErrorChannel <- nil
				fmt.Println("wait group now done")
				wg.Done()
				break
			}
			time.Sleep(time.Millisecond * 200)
		}
	}()

	// wait until the collector server is in the Running state, or an error was returned
	err = <-colErrorChannel
	if err != nil {
		fmt.Println(err)
		return err
	}
	return nil
}

// NopFactories returns a otelcol.Factories with all nop factories.
func NopFactories() (otelcol.Factories, error) {
	var factories otelcol.Factories
	var err error

	if factories.Extensions, err = extension.MakeFactoryMap(extensiontest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Receivers, err = receiver.MakeFactoryMap(receivertest.NewNopFactory()); err != nil {
		return otelcol.Factories{}, err
	}

	if factories.Exporters, err = exporter.MakeFactoryMap(exportertest.NewNopFactory()); err != nil {
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

func makeMapProvidersMap(providers ...confmap.Provider) map[string]confmap.Provider {
	ret := make(map[string]confmap.Provider, len(providers))
	for _, provider := range providers {
		ret[provider.Scheme()] = provider
	}
	return ret
}

type failureProvider struct{}

func newFailureProvider() confmap.Provider {
	return &failureProvider{}
}

func (fmp *failureProvider) Retrieve(context.Context, string, confmap.WatcherFunc) (*confmap.Retrieved, error) {
	return nil, nil
}

func (*failureProvider) Scheme() string {
	return "file"
}

func (*failureProvider) Shutdown(ctx context.Context) error {
	return nil
}
