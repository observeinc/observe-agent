/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package start

import (
	"context"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/cobra"
)

func DefaultLoggerCtx() context.Context {
	return logger.WithCtx(context.Background(), logger.Get())
}

func MakeStartCommand() *cobra.Command {
	otleCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Observe agent process.",
		Long: `The Observe agent is based on the OpenTelemetry Collector.
This command reads in the local config and env vars and starts the
collector on the current host.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			col, cleanup, err := observecol.GetOtelCollector(DefaultLoggerCtx())
			if cleanup != nil {
				defer cleanup()
			}
			if err != nil {
				return err
			}
			return col.Run(cmd.Context())
		},
	}
	return otleCmd
}

func init() {
	startCmd := MakeStartCommand()
	root.RootCmd.AddCommand(startCmd)
}
