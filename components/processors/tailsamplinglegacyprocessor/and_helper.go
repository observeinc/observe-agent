// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tailsamplingprocessor // import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor"

import (
	"go.opentelemetry.io/collector/component"

	"github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/sampling"
	"github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/pkg/samplingpolicy"
)

func getNewAndPolicy(settings component.TelemetrySettings, config *AndCfg, policyExtensions map[string]samplingpolicy.Extension) (samplingpolicy.Evaluator, error) {
	subPolicyEvaluators := make([]samplingpolicy.Evaluator, len(config.SubPolicyCfg))
	for i := range config.SubPolicyCfg {
		policyCfg := &config.SubPolicyCfg[i]
		policy, err := getAndSubPolicyEvaluator(settings, policyCfg, policyExtensions)
		if err != nil {
			return nil, err
		}
		subPolicyEvaluators[i] = policy
	}
	return sampling.NewAnd(settings.Logger, subPolicyEvaluators), nil
}

// Return instance of and sub-policy
func getAndSubPolicyEvaluator(settings component.TelemetrySettings, cfg *AndSubPolicyCfg, policyExtensions map[string]samplingpolicy.Extension) (samplingpolicy.Evaluator, error) {
	return getSharedPolicyEvaluator(settings, &cfg.sharedPolicyCfg, policyExtensions)
}
