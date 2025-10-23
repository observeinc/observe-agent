package start

import (
	"bytes"
	"context"
	"encoding/base64"
	"os"

	"github.com/goccy/go-yaml"
	configcmd "github.com/observeinc/observe-agent/internal/commands/config"
	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func DefaultLoggerCtx() context.Context {
	return logger.WithCtx(context.Background(), logger.Get())
}

func setConfigEnvVars(ctx context.Context) error {
	// Set the observe-agent config
	agentConfig, err := config.AgentConfigFromViper(viper.GetViper())
	if err != nil {
		return err
	}
	if err := agentConfig.Validate(); err != nil {
		return err
	}
	yamlBytes, err := yaml.Marshal(agentConfig)
	if err != nil {
		return err
	}
	// Base64 encode to avoid shell escaping issues
	agentConfigEncoded := base64.StdEncoding.EncodeToString(yamlBytes)
	os.Setenv("OBSERVE_AGENT_CONFIG", agentConfigEncoded)

	// Set the full OTel config
	var output bytes.Buffer
	if err := configcmd.PrintShortOtelConfig(ctx, &output); err != nil {
		return err
	}
	// Base64 encode to avoid shell escaping issues
	otelConfigEncoded := base64.StdEncoding.EncodeToString(output.Bytes())
	os.Setenv("OBSERVE_AGENT_OTEL_CONFIG", otelConfigEncoded)
	return nil
}

func MakeStartCommand() *cobra.Command {
	otleCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the Observe agent process.",
		Long: `The Observe agent is based on the OpenTelemetry Collector.
This command reads in the local config and env vars and starts the
collector on the current host.`,
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			col, cleanup, err := observecol.GetOtelCollector(DefaultLoggerCtx())
			ctx := DefaultLoggerCtx()
			if err := setConfigEnvVars(ctx); err != nil {
				return err
			}
			col, cleanup, err = observecol.GetOtelCollector(ctx)
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
