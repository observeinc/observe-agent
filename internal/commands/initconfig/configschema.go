package initconfig

type HostMonitoringLogsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type HostMonitoringHostMetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type HostMonitoringProcessMetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type HostMonitoringMetricsConfig struct {
	Host    HostMonitoringHostMetricsConfig
	Process HostMonitoringProcessMetricsConfig
}

type HostMonitoringConfig struct {
	Enabled bool `yaml:"enabled"`
	Logs    HostMonitoringLogsConfig
	Metrics HostMonitoringMetricsConfig
}

type SelfMonitoringConfig struct {
	Enabled bool `yaml:"enabled"`
}

type AgentConfig struct {
	Token          string               `yaml:"token"`
	ObserveURL     string               `yaml:"observe_url"`
	SelfMonitoring SelfMonitoringConfig `yaml:"self_monitoring"`
	HostMonitoring HostMonitoringConfig `yaml:"host_monitoring"`
}
