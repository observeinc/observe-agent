package macos

import "embed"

var (
	//go:embed host_monitoring/process_metrics.yaml.tmpl
	ProcessMetricsTemplateFS embed.FS
)
