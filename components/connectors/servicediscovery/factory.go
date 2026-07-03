package servicediscovery

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/connector"
	"go.opentelemetry.io/collector/consumer"
)

var Type = component.MustNewType("servicediscovery")

func NewFactory() connector.Factory {
	return connector.NewFactory(
		Type,
		createDefaultConfig,
		connector.WithTracesToLogs(createTracesToLogsConnector, component.StabilityLevelAlpha),
		connector.WithMetricsToLogs(createMetricsToLogsConnector, component.StabilityLevelAlpha),
		connector.WithLogsToLogs(createLogsToLogsConnector, component.StabilityLevelAlpha))
}

func createDefaultConfig() component.Config {
	return &Config{
		LogExportInterval: 1 * time.Minute,
	}
}

func createTracesToLogsConnector(ctx context.Context, params connector.Settings, cfg component.Config, nextConsumer consumer.Logs) (connector.Traces, error) {
	return newConnector(params.Logger, cfg, nextConsumer)
}

func createMetricsToLogsConnector(ctx context.Context, params connector.Settings, cfg component.Config, nextConsumer consumer.Logs) (connector.Metrics, error) {
	return newConnector(params.Logger, cfg, nextConsumer)
}

func createLogsToLogsConnector(ctx context.Context, params connector.Settings, cfg component.Config, nextConsumer consumer.Logs) (connector.Logs, error) {
	return newConnector(params.Logger, cfg, nextConsumer)
}
