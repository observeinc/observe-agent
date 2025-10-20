package bundledconfig

import (
	"embed"

	"github.com/observeinc/observe-agent/internal/connections/bundledconfig/docker"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig/linux"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig/macos"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig/shared"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig/windows"
)

type ConfigTemplates = map[string]embed.FS

// TODO break up some of the larger connections in order to share more configs.
var SharedTemplateFS = ConfigTemplates{
	"application/RED_metrics.yaml.tmpl":          shared.REDMetrics,
	"common/attributes.yaml.tmpl":                shared.AttributesTemplateFS,
	"common/internal_telemetry.yaml.tmpl":        shared.InternalTelemetryTemplateFS,
	"common/health_check.yaml.tmpl":              shared.HealthCheckTemplateFS,
	"common/base.yaml.tmpl":                      shared.BaseTemplateFS,
	"common/extensions.yaml.tmpl":                shared.ExtensionsTemplateFS,
	"common/forward.yaml.tmpl":                   shared.ForwardTemplateFS,
	"common/resource_detection.yaml.tmpl":        shared.ResourceDetectionTemplateFS,
	"host_monitoring/logs.yaml.tmpl":             shared.LogsTemplateFS,
	"host_monitoring/host_metrics.yaml.tmpl":     shared.HostMetricsTemplateFS,
	"host_monitoring/host.yaml.tmpl":             shared.HostTemplateFS,
	"host_monitoring/process_metrics.yaml.tmpl":  shared.ProcessMetricsTemplateFS,
	"self_monitoring/logs_and_metrics.yaml.tmpl": shared.LogsAndMetricsTemplateFS,
	"fleet/heartbeat_shared.yaml.tmpl":           shared.HeartbeatSharedTemplateFS,
}

var DockerTemplateFS = ConfigTemplates{
	"common/extensions.yaml.tmpl":                docker.ExtensionsTemplateFS,
	"host_monitoring/logs.yaml.tmpl":             docker.LogsTemplateFS,
	"host_monitoring/host_metrics.yaml.tmpl":     docker.HostMetricsTemplateFS,
	"host_monitoring/process_metrics.yaml.tmpl":  docker.ProcessMetricsTemplateFS,
	"self_monitoring/logs_and_metrics.yaml.tmpl": docker.LogsAndMetricsTemplateFS,
	"fleet/heartbeat_receiver.yaml.tmpl":         docker.HeartbeatTemplateFS,
}

var LinuxTemplateFS = ConfigTemplates{
	"host_monitoring/logs.yaml.tmpl":             linux.LogsTemplateFS,
	"host_monitoring/host_metrics.yaml.tmpl":     linux.HostMetricsTemplateFS,
	"self_monitoring/logs_and_metrics.yaml.tmpl": linux.LogsAndMetricsTemplateFS,
	"fleet/heartbeat_receiver.yaml.tmpl":         linux.HeartbeatTemplateFS,
}

var MacOSTemplateFS = ConfigTemplates{
	"fleet/heartbeat_receiver.yaml.tmpl": macos.HeartbeatTemplateFS,
}

var WindowsTemplateFS = ConfigTemplates{
	"common/resource_detection.yaml.tmpl":    windows.ResourceDetectionTemplateFS,
	"host_monitoring/logs.yaml.tmpl":         windows.LogsTemplateFS,
	"host_monitoring/host_metrics.yaml.tmpl": windows.HostMetricsTemplateFS,
	"fleet/heartbeat_receiver.yaml.tmpl":     windows.HeartbeatTemplateFS,
}
