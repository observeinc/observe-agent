package status

import (
	"github.com/spf13/viper"

	"github.com/spf13/cobra"
)

const (
	TelemetryEndpointFlag   = "telemetry-endpoint"
	HealthcheckEndpointFlag = "healthcheck-endpoint"
	HealthcheckPathFlag     = "healthcheck-path"
)

var (
	telemetryEndpoint   string
	healthcheckEndpoint string
	healthcheckPath     string
)

func RegisterStatusFlags(cmd *cobra.Command, v *viper.Viper) {
	// Use persistent flags and fixed variables so these flags can be used in multiple subcommands
	pf := cmd.PersistentFlags()
	pf.StringVar(&telemetryEndpoint, TelemetryEndpointFlag, "http://localhost:8888", "Endpoint the observe-agent has exposed for internal telemetry data")
	pf.StringVar(&healthcheckEndpoint, HealthcheckEndpointFlag, "http://localhost:13133", "Endpoint the observe-agent has configured for the health check connector")
	pf.StringVar(&healthcheckPath, HealthcheckPathFlag, "/status", "Path the observe-agent has configured for the health check connector")
	v.BindPFlag(TelemetryEndpointFlag, pf.Lookup(TelemetryEndpointFlag))
	v.BindPFlag(HealthcheckEndpointFlag, pf.Lookup(HealthcheckEndpointFlag))
	v.BindPFlag(HealthcheckPathFlag, pf.Lookup(HealthcheckPathFlag))
}
