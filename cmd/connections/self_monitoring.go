package connections

type SelfMonitoringConfig struct {
	enabled bool
}

var SelfMonitoringConnectionType = ConnectionType{
	Name: "self_monitoring",
	ConfigFields: []CollectorConfigFragment{
		{
			configYAMLPath:    "enabled",
			colConfigFilePath: "logs_and_metrics.yaml",
		},
	},
}
