/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package start

import (
	"observe/agent/cmd"
	observeotel "observe/otelcol"
	"observe/agent/cmd/config"
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
		// Set Env Vars from config
		err := config.SetEnvVars()
		if err != nil {
			return err
		}
		//
		configFilePaths, overridePath, err := config.GetAllOtelConfigFilePaths()
		if err != nil {
			return err
		}
		if overridePath != "" {
			defer os.Remove(overridePath)
		}
		// Generate collector settings with all config files
		colSettings := observeotel.GenerateCollectorSettings(configFilePaths)
		otelCmd := observeotel.GetOtelCollectorCommand(colSettings)
		return otelCmd.RunE(cmd, args)
	},
}

func init() {
	startCmd.PersistentFlags().String("otel-config", "", "Path to additional otel configuration file")
	viper.BindPFlag("otelConfigFile", startCmd.PersistentFlags().Lookup("otel-config"))
	cmd.RootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
