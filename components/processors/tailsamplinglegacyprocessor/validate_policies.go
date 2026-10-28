package tailsamplingprocessor // import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor"

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

// ValidatePolicies builds every sampling policy in cfg the way Start does, without starting the
// processor. Policies provided by extensions need a running host, so they are reported as errors.
func ValidatePolicies(cfg component.Config, set component.TelemetrySettings) error {
	c, ok := cfg.(*Config)
	if !ok {
		return fmt.Errorf("unexpected config type %T", cfg)
	}
	for i := range c.PolicyCfgs {
		p := &c.PolicyCfgs[i]
		if _, err := getPolicyEvaluator(set, p, nil); err != nil {
			return fmt.Errorf("policy %q: %w", p.Name, err)
		}
	}
	return nil
}
