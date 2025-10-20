package heartbeatreceiver

import (
	"context"

	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

type ReceiverType struct{}

func (ReceiverType) Type() component.Type {
	return metadata.Type
}

const (
	defaultIntervalString       = "10m"
	defaultConfigIntervalString = "24h"
	defaultEnvironment          = "linux"
)

func createDefaultConfig() component.Config {
	return &Config{
		Interval:       defaultIntervalString,
		ConfigInterval: defaultConfigIntervalString,
		Environment:    defaultEnvironment,
	}
}

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

func createLogsReceiver(
	ctx context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	c := cfg.(*Config)
	return newReceiver(set, c, nextConsumer)
}
