/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/observeinc/observe-agent/internal/commands/start"
	logger "github.com/observeinc/observe-agent/internal/commands/util"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Prints the full configuration for this agent.",
	Long: `This command prints all configuration for this agent including any additional
bundled OTel configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		detailedOtel, err := cmd.Flags().GetBool("render-otel-details")
		if err != nil {
			return err
		}
		singleOtel, err := cmd.Flags().GetBool("render-otel")
		if err != nil {
			return err
		}
		if singleOtel && detailedOtel {
			return fmt.Errorf("cannot specify both --render-otel and --render-otel-details")
		}

		ctx := logger.WithCtx(context.Background(), logger.GetNop())
		configFilePaths, cleanup, err := start.SetupAndGetConfigFiles(ctx)
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			return err
		}
		if singleOtel {
			return printShortOtelConfig(ctx, configFilePaths)
		} else if detailedOtel {
			return printFullOtelConfig(configFilePaths)
		}
		return printAllConfigsIndividually(configFilePaths)
	},
}

func printAllConfigsIndividually(configFilePaths []string) error {
	printConfig := func(comment string, data []byte) {
		fmt.Printf("# ======== %s\n", comment)
		fmt.Println(strings.Trim(string(data), "\n\t "))
		fmt.Println("---")
	}

	agentConfig, err := config.AgentConfigFromViper(viper.GetViper())
	if err != nil {
		return err
	}
	agentConfigYaml, err := yaml.Marshal(agentConfig)
	if err != nil {
		return err
	}
	printConfig("computed agent config", agentConfigYaml)
	agentConfigFile := viper.ConfigFileUsed()
	if agentConfigFile != "" {
		configFilePaths = append([]string{agentConfigFile}, configFilePaths...)
	}
	for _, filePath := range configFilePaths {
		file, err := os.ReadFile(filePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading config file %s: %s", filePath, err.Error())
		} else {
			printConfig("config file "+filePath, file)
		}
	}
	return nil
}

func printShortOtelConfig(ctx context.Context, configFilePaths []string) error {
	settings := observecol.ConfigProviderSettings(configFilePaths)
	resolver, err := confmap.NewResolver(settings.ResolverSettings)
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
	fmt.Printf("%s\n", b)
	return nil
}

func printFullOtelConfig(configFilePaths []string) error {
	colSettings := observecol.GenerateCollectorSettingsWithConfigFiles(configFilePaths)
	factories, err := colSettings.Factories()
	if err != nil {
		return fmt.Errorf("failed to create component factory maps: %w", err)
	}
	provider, err := otelcol.NewConfigProvider(colSettings.ConfigProviderSettings)
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
	fmt.Printf("%s\n", cfgYaml)
	return nil
}

func init() {
	configCmd.Flags().Bool("render-otel-details", false, "Print the full resolved otel configuration including default values after the otel components perform their semantic processing.")
	configCmd.Flags().Bool("render-otel", false, "Print a single rendered otel configuration file. This file is equivalent to the bundled configuration enabled in the observe-agent config.")
	root.RootCmd.AddCommand(configCmd)
}
