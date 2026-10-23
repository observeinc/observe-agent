package start

import (
	"bytes"
	"context"
	"os"

	"github.com/goccy/go-yaml"
	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver"
	configcmd "github.com/observeinc/observe-agent/internal/commands/config"
	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func DefaultLoggerCtx() context.Context {
	return logger.WithCtx(context.Background(), logger.Get())
}

// setHeartbeatConfigs hands the resolved configs to the heartbeat receiver in-process. They must
// not be put in env vars: child processes (e.g. journalctl) inherit the environment, and a single
// env string over 128 KiB makes execve fail with E2BIG ("argument list too long").
func setHeartbeatConfigs(ctx context.Context) error {
	// Clear inherited legacy values so they cannot reach child processes.
	os.Unsetenv("OBSERVE_AGENT_CONFIG")
	os.Unsetenv("OBSERVE_AGENT_OTEL_CONFIG")

	// Set the observe-agent config
	agentConfig, err := config.AgentConfigFromViper(viper.GetViper())
	if err != nil {
		// Mark configs as set so the receiver skips the env var fallback and its misleading errors.
		heartbeatreceiver.SetConfigs("", "")
		return err
	}
	if err := agentConfig.Validate(); err != nil {
		logger.FromCtx(ctx).Error("failed to set validate observe-agent config, attempting to start anyway", zap.Error(err))
	}
	yamlBytes, err := yaml.Marshal(agentConfig)
	if err != nil {
		heartbeatreceiver.SetConfigs("", "")
		return err
	}

	// Set the full OTel config
	var output bytes.Buffer
	if err := configcmd.PrintShortOtelConfig(ctx, &output); err != nil {
		// Still send a partial heartbeat with the agent config; Fleet needs it most when
		// rendering fails.
		heartbeatreceiver.SetConfigs(string(yamlBytes), "")
		return err
	}
	heartbeatreceiver.SetConfigs(string(yamlBytes), output.String())
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
			ctx := DefaultLoggerCtx()
			if err := setHeartbeatConfigs(ctx); err != nil {
				logger.FromCtx(ctx).Error("failed to set observe-agent heartbeat configs, attempting to start anyway", zap.Error(err))
			}
			col, err := observecol.GetOtelCollector(ctx)
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
