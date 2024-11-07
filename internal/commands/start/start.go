/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package start

import (
	"context"
	"os"

	logger "github.com/observeinc/observe-agent/internal/commands/util"
	"github.com/observeinc/observe-agent/internal/connections"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	collector "go.opentelemetry.io/collector/otelcol"
)

func SetupAndGetConfigFiles(ctx context.Context) ([]string, func(), error) {
	// Set Env Vars from config
	err := connections.SetEnvVars()
	if err != nil {
		return nil, nil, err
	}
	// Set up our temp dir annd temp config files
	tmpDir, err := os.MkdirTemp("", connections.TempFilesFolder)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	configFilePaths, err := connections.GetAllOtelConfigFilePaths(ctx, tmpDir)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return configFilePaths, cleanup, nil
}

func DefaultLoggerCtx() context.Context {
	return logger.WithCtx(context.Background(), logger.Get())
}

func SetupAndGenerateCollectorSettings(ctx context.Context) (*collector.CollectorSettings, func(), error) {
	configFilePaths, cleanup, err := SetupAndGetConfigFiles(ctx)
	if err != nil {
		return nil, cleanup, err
	}
	// Generate collector settings with all config files
	colSettings := observecol.GenerateCollectorSettings(configFilePaths)
	return colSettings, cleanup, nil
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Observe agent process.",
	Long: `The Observe agent is based on the OpenTelemetry Collector. 
This command reads in the local config and env vars and starts the 
collector on the current host.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		colSettings, cleanup, err := SetupAndGenerateCollectorSettings(DefaultLoggerCtx())
		if err != nil {
			return err
		}
		if cleanup != nil {
			defer cleanup()
		}
		otelCmd := observecol.GetOtelCollectorCommand(colSettings)
		return otelCmd.RunE(cmd, args)
	},
}

func init() {
	startCmd.PersistentFlags().String("otel-config", "", "Path to additional otel configuration file")
	viper.BindPFlag("otelConfigFile", startCmd.PersistentFlags().Lookup("otel-config"))
	root.RootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
