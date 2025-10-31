package docker

import "embed"

var (
	//go:embed common/extensions.yaml.tmpl
	ExtensionsTemplateFS embed.FS
	//go:embed host_monitoring/logs.yaml.tmpl
	LogsTemplateFS embed.FS
	//go:embed host_monitoring/host_metrics.yaml.tmpl
	HostMetricsTemplateFS embed.FS
	//go:embed host_monitoring/process_metrics.yaml.tmpl
	ProcessMetricsTemplateFS embed.FS
	//go:embed self_monitoring/logs_and_metrics.yaml.tmpl
	LogsAndMetricsTemplateFS embed.FS
)
