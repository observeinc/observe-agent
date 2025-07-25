package bundledconfig

import (
	"embed"

	"github.com/observeinc/observe-agent/internal/connections/bundledconfig/docker"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig/linux"
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
	"common/forward.yaml.tmpl":                   shared.ForwardTemplateFS,
	"host_monitoring/logs.yaml.tmpl":             shared.LogsTemplateFS,
	"host_monitoring/host_metrics.yaml.tmpl":     shared.HostMetricsTemplateFS,
	"host_monitoring/host.yaml.tmpl":             shared.HostTemplateFS,
	"host_monitoring/process_metrics.yaml.tmpl":  shared.ProcessMetricsTemplateFS,
	"self_monitoring/logs_and_metrics.yaml.tmpl": shared.LogsAndMetricsTemplateFS,
}

var DockerTemplateFS = ConfigTemplates{
	"common/base.yaml.tmpl":                      docker.BaseTemplateFS,
	"host_monitoring/logs.yaml.tmpl":             docker.LogsTemplateFS,
	"host_monitoring/host_metrics.yaml.tmpl":     docker.HostMetricsTemplateFS,
	"host_monitoring/process_metrics.yaml.tmpl":  docker.ProcessMetricsTemplateFS,
	"self_monitoring/logs_and_metrics.yaml.tmpl": docker.LogsAndMetricsTemplateFS,
}

var LinuxTemplateFS = ConfigTemplates{
	"host_monitoring/logs.yaml.tmpl":             linux.LogsTemplateFS,
	"host_monitoring/host_metrics.yaml.tmpl":     linux.HostMetricsTemplateFS,
	"self_monitoring/logs_and_metrics.yaml.tmpl": linux.LogsAndMetricsTemplateFS,
}

var MacOSTemplateFS = ConfigTemplates{}

var WindowsTemplateFS = ConfigTemplates{
	"common/base.yaml.tmpl":                  windows.BaseTemplateFS,
	"host_monitoring/logs.yaml.tmpl":         windows.LogsTemplateFS,
	"host_monitoring/host_metrics.yaml.tmpl": windows.HostMetricsTemplateFS,
}
