package connections

var SelfMonitoringConnectionTypeName = "self_monitoring"

type SelfMonitoringConfig struct {
	enabled bool
}

var SelfMonitoringConnectionType = MakeConnectionType(
	"self_monitoring",
	[]CollectorConfigFragment{
		{
			configYAMLPath:    "enabled",
			colConfigFilePath: "logs_and_metrics.yaml",
		},
	},
	SelfMonitoringConnectionTypeName,
)
