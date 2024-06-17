package initconfig

type HostMonitoringLogsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type HostMonitoringMetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type HostMonitoringConfig struct {
	Enabled bool `yaml:"enabled"`
	Logs    HostMonitoringLogsConfig
	Metrics HostMonitoringMetricsConfig
}

type AgentConfig struct {
	Token          string               `yaml:"token"`
	ObserveURL     string               `yaml:"observe_url"`
	HostMonitoring HostMonitoringConfig `yaml:"host_monitoring"`
}
