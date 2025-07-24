package shared

import "embed"

var (
	//go:embed common/attributes.yaml.tmpl
	AttributesTemplateFS embed.FS
	//go:embed common/internal_telemetry.yaml.tmpl
	InternalTelemetryTemplateFS embed.FS
	//go:embed common/health_check.yaml.tmpl
	HealthCheckTemplateFS embed.FS
	//go:embed common/base.yaml.tmpl
	BaseTemplateFS embed.FS
	//go:embed common/forward.yaml.tmpl
	ForwardTemplateFS embed.FS
	//go:embed host_monitoring/logs.yaml.tmpl
	LogsTemplateFS embed.FS
	//go:embed host_monitoring/host_metrics.yaml.tmpl
	HostMetricsTemplateFS embed.FS
	//go:embed host_monitoring/host.yaml.tmpl
	HostTemplateFS embed.FS
	//go:embed host_monitoring/process_metrics.yaml.tmpl
	ProcessMetricsTemplateFS embed.FS
	//go:embed self_monitoring/logs_and_metrics.yaml.tmpl
	LogsAndMetricsTemplateFS embed.FS
	//go:embed application/RED_metrics.yaml.tmpl
	REDMetrics embed.FS
)
