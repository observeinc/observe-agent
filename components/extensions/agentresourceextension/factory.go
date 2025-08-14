package agentresourceextension

import (
	"context"

	"github.com/observeinc/observe-agent/components/extensions/agentresourceextension/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/extension"
)

func NewFactory() extension.Factory {
	return extension.NewFactory(
		metadata.Type,
		createDefaultConfig,
		createExtension,
		metadata.ExtensionStability,
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		LocalFilePath: GetDefaultAgentPath() + "/agent_local_data.json",
	}
}

func createExtension(
	ctx context.Context,
	set extension.Settings,
	cfg component.Config,
) (extension.Extension, error) {
	config := cfg.(*Config)
	return newAgentResourceExtension(config, set.Logger), nil
}
