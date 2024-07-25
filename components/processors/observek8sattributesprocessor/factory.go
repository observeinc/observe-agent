package observek8sattributesprocessor

import (
	"context"

	"github.com/observeinc/observe-agent/components/processors/observek8sattributesprocessor/internal/metadata"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithLogs(createLogsProcessor, metadata.LogsStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createLogsProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (processor.Logs, error) {
	kep := newK8sEventsProcessor(set.Logger, cfg)
	return processorhelper.NewLogsProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		kep.processLogs,
		processorhelper.WithCapabilities(processorCapabilities),
		processorhelper.WithStart(kep.Start),
		processorhelper.WithShutdown(kep.Shutdown),
	)
}
