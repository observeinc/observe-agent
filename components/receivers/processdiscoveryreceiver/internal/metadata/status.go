package metadata

import "go.opentelemetry.io/collector/component"

var (
	Type      = component.MustNewType("processdiscovery")
	ScopeName = "processdiscoveryreceiver"
)

const (
	LogsStability = component.StabilityLevelDevelopment
)
