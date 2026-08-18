//go:build !linux

package processdiscoveryreceiver

func detectOTLPConnections(_ string, _ int32, _ []otlpEndpoint, _ int) InstrumentationEvidence {
	return InstrumentationEvidence{Status: "unavailable"}
}

type otlpEndpoint struct {
	Host string
	Port int
}
