package connections

type HostMonitoringConfig struct {
	enabled bool
	metrics struct {
		enabled bool
	}
	logs struct {
		enabled bool
	}
}

var HostMonitoringConnectionType = ConnectionType{
	Name: "host_monitoring",
	ConfigFields: []CollectorConfigFragment{
		{
			configYAMLPath:    "metrics.enabled",
			colConfigFilePath: "/etc/observe-agent/hostmonitoring_metrics.yaml",
		},
		{
			configYAMLPath:    "logs.enabled",
			colConfigFilePath: "/etc/observe-agent/hostmonitoring_logs.yaml",
		},
	},
}