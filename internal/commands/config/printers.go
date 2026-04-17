package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/connections"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"gopkg.in/yaml.v3"
)

func PrintAllConfigsIndividually(ctx context.Context, w io.Writer) error {
	fragments, err := connections.SetupAndGetConfigs(ctx)
	if err != nil {
		return err
	}

	printConfig := func(comment string, data []byte) {
		fmt.Fprintf(w, "# ======== %s\n", comment)
		fmt.Fprintln(w, strings.Trim(string(data), "\n\t "))
		fmt.Fprintln(w, "---")
	}

	agentConfig, err := config.AgentConfigFromViper(viper.GetViper())
	if err != nil {
		return err
	}
	// Use mapstructure as an intermediary so all values are printed.
	var agentConfigMap map[string]any
	err = mapstructure.Decode(agentConfig, &agentConfigMap)
	if err != nil {
		return err
	}
	agentConfigYaml, err := yaml.Marshal(&agentConfigMap)
	if err != nil {
		return err
	}
	printConfig("computed agent config", agentConfigYaml)
	if agentConfigFile := viper.ConfigFileUsed(); agentConfigFile != "" {
		file, err := os.ReadFile(agentConfigFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading config file %s: %s", agentConfigFile, err.Error())
		} else {
			printConfig("config file "+agentConfigFile, file)
		}
	}
	for _, f := range fragments {
		printConfig("config fragment "+f.Name, []byte(f.Content))
	}
	return nil
}

func PrintShortOtelConfig(ctx context.Context, w io.Writer) error {
	settings, err := observecol.GetOtelCollectorSettings(ctx)
	if err != nil {
		return err
	}
	if len(settings.ConfigProviderSettings.ResolverSettings.URIs) == 0 {
		return nil
	}
	resolver, err := confmap.NewResolver(settings.ConfigProviderSettings.ResolverSettings)
	if err != nil {
		return fmt.Errorf("failed to create new resolver: %w", err)
	}
	conf, err := resolver.Resolve(ctx)
	if err != nil {
		return fmt.Errorf("error while resolving config: %w", err)
	}
	b, err := yaml.Marshal(conf.ToStringMap())
	if err != nil {
		return fmt.Errorf("error while marshaling to YAML: %w", err)
	}
	_, err = w.Write(b)
	return err
}

func PrintFullOtelConfig(ctx context.Context, w io.Writer) error {
	settings, err := observecol.GetOtelCollectorSettings(ctx)
	if err != nil {
		return err
	}
	if len(settings.ConfigProviderSettings.ResolverSettings.URIs) == 0 {
		return nil
	}
	factories, err := settings.Factories()
	if err != nil {
		return fmt.Errorf("failed to create component factory maps: %w", err)
	}
	provider, err := otelcol.NewConfigProvider(settings.ConfigProviderSettings)
	if err != nil {
		return fmt.Errorf("failed to create config provider: %w", err)
	}
	cfg, err := provider.Get(context.Background(), factories)
	if err != nil {
		return fmt.Errorf("failed to get config: %w", err)
	}
	var cfgMap map[string]any
	err = mapstructure.Decode(cfg, &cfgMap)
	if err != nil {
		return fmt.Errorf("failed to marshall config to map: %w", err)
	}
	cfgYaml, err := yaml.Marshal(cfgMap)
	if err != nil {
		return fmt.Errorf("failed to marshall config to yaml: %w", err)
	}
	fmt.Fprintf(w, "%s\n", cfgYaml)
	return nil
}
