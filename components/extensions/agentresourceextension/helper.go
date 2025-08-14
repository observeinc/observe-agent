package agentresourceextension

import (
	"fmt"

	"github.com/observeinc/observe-agent/components/extensions/agentresourceextension/internal/metadata"
	"go.opentelemetry.io/collector/component"
)

func GetAgentResourceProvider(host component.Host) (AgentResourceProvider, error) {
	ext, found := host.GetExtensions()[component.NewID(metadata.Type)]
	if !found {
		return nil, fmt.Errorf("agentresource extension not found")
	}

	provider, ok := ext.(AgentResourceProvider)
	if !ok {
		return nil, fmt.Errorf("extension does not implement AgentResourceProvider interface")
	}

	return provider, nil
}