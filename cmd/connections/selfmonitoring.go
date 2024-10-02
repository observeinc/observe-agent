package connections

var SelfMonitoringConnectionTypeName = "self_monitoring"

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
	Type: SelfMonitoringConnectionTypeName,
}
