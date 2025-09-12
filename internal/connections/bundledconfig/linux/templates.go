package linux

import "embed"

var (
	//go:embed host_monitoring/logs.yaml.tmpl
	LogsTemplateFS embed.FS
	//go:embed host_monitoring/host_metrics.yaml.tmpl
	HostMetricsTemplateFS embed.FS
	//go:embed self_monitoring/logs_and_metrics.yaml.tmpl
	LogsAndMetricsTemplateFS embed.FS
	//go:embed fleet/heartbeat_receiver.yaml.tmpl
	HeartbeatTemplateFS embed.FS
)
