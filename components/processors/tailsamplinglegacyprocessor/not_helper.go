// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tailsamplingprocessor // import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor"

import (
	"go.opentelemetry.io/collector/component"

	"github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/sampling"
	"github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/pkg/samplingpolicy"
)

func getNewNotPolicy(settings component.TelemetrySettings, config *NotCfg, policyExtensions map[string]samplingpolicy.Extension) (samplingpolicy.Evaluator, error) {
	subPolicyEvaluator, err := getNotSubPolicyEvaluator(settings, &config.SubPolicy, policyExtensions)
	if err != nil {
		return nil, err
	}
	return sampling.NewNot(settings.Logger, subPolicyEvaluator), nil
}

// Return instance of not sub-policy
func getNotSubPolicyEvaluator(settings component.TelemetrySettings, cfg *NotSubPolicyCfg, policyExtensions map[string]samplingpolicy.Extension) (samplingpolicy.Evaluator, error) {
	return getSharedPolicyEvaluator(settings, &cfg.sharedPolicyCfg, policyExtensions)
}
