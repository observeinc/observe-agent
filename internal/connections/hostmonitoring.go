package connections

var HostMonitoringConnectionTypeName = "host_monitoring"

type HostMonitoringConfig struct {
	enabled bool
	metrics struct {
		enabled bool
	}
	logs struct {
		enabled bool
	}
}

var HostMonitoringConnectionType = MakeConnectionType(
	"host_monitoring",
	[]CollectorConfigFragment{
		{
			configYAMLPath:    "enabled",
			colConfigFilePath: "host.yaml",
		},
		{
			configYAMLPath:    "metrics::enabled",
			colConfigFilePath: "host_metrics.yaml",
		},
		{
			configYAMLPath:    "metrics::host::enabled",
			colConfigFilePath: "host_metrics.yaml",
		},
		{
			configYAMLPath:    "metrics::process::enabled",
			colConfigFilePath: "process_metrics.yaml",
		},
		{
			configYAMLPath:    "logs::enabled",
			colConfigFilePath: "logs.yaml",
		},
	},
	HostMonitoringConnectionTypeName,
)