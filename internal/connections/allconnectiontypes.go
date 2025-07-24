package connections

import (
	"github.com/observeinc/observe-agent/internal/config"
)

var AllConnectionTypes = []*ConnectionType{
	CommonConnectionType,
	HostMonitoringConnectionType,
	SelfMonitoringConnectionType,
	ApplicationConnectionType,
}

var CommonConnectionType = MakeConnectionType(
	"common",
	func(_ *config.AgentConfig) bool {
		return true
	},
	[]BundledConfigFragment{
		{
			enabledCheck: func(_ *config.AgentConfig) bool {
				// Always include the base connection.
				return true
			},
			colConfigFilePath: "base.yaml.tmpl",
		},
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.HasAttributes() || agentConfig.HasResourceAttributes()
			},
			colConfigFilePath: "attributes.yaml.tmpl",
		},
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.Forwarding.Enabled
			},
			colConfigFilePath: "forward.yaml.tmpl",
		},
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.HealthCheck.Enabled
			},
			colConfigFilePath: "health_check.yaml.tmpl",
		},
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.InternalTelemetry.Enabled
			},
			colConfigFilePath: "internal_telemetry.yaml.tmpl",
		},
	},
)

var HostMonitoringConnectionType = MakeConnectionType(
	"host_monitoring",
	func(agentConfig *config.AgentConfig) bool {
		return agentConfig.HostMonitoring.Enabled
	},
	[]BundledConfigFragment{
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				// TODO remove this deprecated template
				return agentConfig.HostMonitoring.Enabled
			},
			colConfigFilePath: "host.yaml.tmpl",
		},
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.HostMonitoring.Metrics.Host.Enabled
			},
			colConfigFilePath: "host_metrics.yaml.tmpl",
		},
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.HostMonitoring.Metrics.Process.Enabled
			},
			colConfigFilePath: "process_metrics.yaml.tmpl",
		},
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.HostMonitoring.Logs.Enabled
			},
			colConfigFilePath: "logs.yaml.tmpl",
		},
	},
)

var SelfMonitoringConnectionType = MakeConnectionType(
	"self_monitoring",
	func(agentConfig *config.AgentConfig) bool {
		return agentConfig.SelfMonitoring.Enabled
	},
	[]BundledConfigFragment{
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.SelfMonitoring.Enabled
			},
			colConfigFilePath: "logs_and_metrics.yaml.tmpl",
		},
	},
)

var ApplicationConnectionType = MakeConnectionType(
	"application",
	func(agentConfig *config.AgentConfig) bool {
		// Make this check more broadly applicable when we have more than one application connection type.
		return agentConfig.Application.REDMetrics.Enabled
	},
	[]BundledConfigFragment{
		{
			enabledCheck: func(agentConfig *config.AgentConfig) bool {
				return agentConfig.Application.REDMetrics.Enabled
			},
			colConfigFilePath: "RED_metrics.yaml.tmpl",
		},
	},
)
