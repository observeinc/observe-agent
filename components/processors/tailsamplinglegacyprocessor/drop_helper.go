// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tailsamplingprocessor // import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor"

import (
	"go.opentelemetry.io/collector/component"

	"github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/sampling"
	"github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/pkg/samplingpolicy"
)

func getNewDropPolicy(settings component.TelemetrySettings, config *DropCfg, policyExtensions map[string]samplingpolicy.Extension) (samplingpolicy.Evaluator, error) {
	subPolicyEvaluators := make([]samplingpolicy.Evaluator, len(config.SubPolicyCfg))
	for i := range config.SubPolicyCfg {
		policyCfg := &config.SubPolicyCfg[i]
		policy, err := getDropSubPolicyEvaluator(settings, policyCfg, policyExtensions)
		if err != nil {
			return nil, err
		}
		subPolicyEvaluators[i] = policy
	}
	return sampling.NewDrop(settings.Logger, subPolicyEvaluators), nil
}

// Return instance of and sub-policy
func getDropSubPolicyEvaluator(settings component.TelemetrySettings, cfg *AndSubPolicyCfg, policyExtensions map[string]samplingpolicy.Extension) (samplingpolicy.Evaluator, error) {
	return getSharedPolicyEvaluator(settings, &cfg.sharedPolicyCfg, policyExtensions)
}
