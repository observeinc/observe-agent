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

func MakeStartCommand() *cobra.Command {
	// Create the start command from the otel collector command
	settings := observecol.GenerateCollectorSettings()
	otleCmd := observecol.GetOtelCollectorCommand(settings)
	otleCmd.Use = "start"
	otleCmd.Short = "Start the Observe agent process."
	otleCmd.Long = `The Observe agent is based on the OpenTelemetry Collector.
This command reads in the local config and env vars and starts the
collector on the current host.`
	// Drop the sub commands
	otleCmd.ResetCommands()

	// Modify the run function so we can pass in our packaged config files.
	originalRunE := otleCmd.RunE
	otleCmd.RunE = func(cmd *cobra.Command, args []string) error {
		configFilePaths, cleanup, err := SetupAndGetConfigFiles(DefaultLoggerCtx())
		if cleanup != nil {
			defer cleanup()
		}
		if err != nil {
			return err
		}
		configFlag := otleCmd.Flags().Lookup("config")
		for _, path := range configFilePaths {
			configFlag.Value.Set(path)
		}
		return originalRunE(cmd, args)
	}
	return otleCmd
}

func init() {
	startCmd := MakeStartCommand()
	root.RootCmd.AddCommand(startCmd)
}
