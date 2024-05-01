/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package start

import (
	"observe/agent/cmd"
	observeotel "observe/agent/cmd/collector"
	"observe/agent/cmd/connections"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Observe agent process.",
	Long: `The Observe agent is based on the OpenTelemetry Collector. 
This command reads in the local config and env vars and starts the 
collector on the current host.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Handle base OTEL config
		err := observeotel.SetEnvVars()
		if err != nil {
			return err
		}
		// Initialize config file paths with base config
		configFilePaths := []string{observeotel.BaseOtelCollectorConfigFilePath}
		// Get additional config paths based on connection configs
		if viper.IsSet(connections.HostMonitoringConnectionType.Name) {
			configFilePaths = append(configFilePaths, connections.HostMonitoringConnectionType.GetConfigFilePaths()...)
		}
		// Generate override file and include path if overrides provided
		var overridePath string
		if viper.IsSet("otel_config_overrides") {
			overridePath, err = observeotel.GetOverrideConfigFile(viper.Sub("otel_config_overrides"))
			if err != nil {
				return err
			}
			configFilePaths = append(configFilePaths, overridePath)
		}
		defer func() {
			if viper.IsSet("otel_config_overrides") {
				os.Remove(overridePath)
			}
		}()
		// Generate collector settings with all config files
		colSettings, err := observeotel.GenerateCollectorSettings(configFilePaths)
		if err != nil {
			return err
		}
		otelCmd := observeotel.GetOtelCollectorCommand(colSettings)
		return otelCmd.RunE(cmd, args)
	},
}

func init() {
	cmd.RootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
