/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"context"
	"fmt"
	"os"

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
OTEL configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logger.WithCtx(context.Background(), logger.GetNop())
		configFilePaths, cleanup, err := start.SetupAndGetConfigFiles(ctx)
		if err != nil {
			return err
		}
		if cleanup != nil {
			defer cleanup()
		}
		agentConfig, err := config.AgentConfigFromViper(viper.GetViper())
		if err != nil {
			return err
		}
		agentConfigYaml, err := yaml.Marshal(agentConfig)
		if err != nil {
			return err
		}
		fmt.Printf("# ======== computed agent config\n")
		fmt.Println(string(agentConfigYaml) + "\n")
		agentConfigFile := viper.ConfigFileUsed()
		if agentConfigFile != "" {
			configFilePaths = append([]string{agentConfigFile}, configFilePaths...)
		}
		for _, filePath := range configFilePaths {
			file, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading config file %s: %s", filePath, err.Error())
			} else {
				fmt.Printf("# ======== config file %s\n", filePath)
				fmt.Println(string(file))
			}
		}
		return nil
	},
}

var otelConfigSubCmd = &cobra.Command{
	Use:   "export-otel",
	Short: "Prints a single otel config file containing the full configuration that would run.",
	Long: `This command prints a single otel config file containing all bundled configuration.
Features that are enabled or disabled in the observe-agent config will be reflected in the output accordingly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configFilePaths, cleanup, err := start.SetupAndGetConfigFiles(logger.WithCtx(context.Background(), logger.GetNop()))
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			return err
		}
		fullVersion, err := cmd.Flags().GetBool("full")
		if err != nil {
			return err
		}
		if fullVersion {
			return printFullOtelConfig(configFilePaths)
		}
		return printShortOtelConfig(cmd.Context(), configFilePaths)
	},
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
		return fmt.Errorf("failed to initialize factories: %w", err)
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
	otelConfigSubCmd.Flags().Bool("full", false, "Print the full resolved configuration including default values instead of the pre-processed configuration.")
	configCmd.AddCommand(otelConfigSubCmd)
	root.RootCmd.AddCommand(configCmd)
}
