/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"context"
	"fmt"
	"os"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/cobra"
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
		if singleOtel {
			return PrintShortOtelConfig(ctx, os.Stdout)
		} else if detailedOtel {
			return PrintFullOtelConfig(ctx, os.Stdout)
		}
		return PrintAllConfigsIndividually(ctx, os.Stdout)
	},
}

func init() {
	configCmd.Flags().Bool("render-otel-details", false, "Print the full resolved otel configuration including default values after the otel components perform their semantic processing.")
	configCmd.Flags().Bool("render-otel", false, "Print a single rendered otel configuration file. This file is equivalent to the bundled configuration enabled in the observe-agent config.")
	root.RootCmd.AddCommand(configCmd)
}
