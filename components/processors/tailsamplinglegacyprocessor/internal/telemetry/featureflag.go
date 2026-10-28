// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package telemetry // import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/telemetry"

import (
	"github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/metadata"
)

func IsMetricStatCountSpansSampledEnabled() bool {
	return metadata.ProcessorTailsamplingprocessorMetricstatcountspanssampledFeatureGate.IsEnabled()
}

func IsRecordPolicyEnabled() bool {
	return metadata.ProcessorTailsamplingprocessorRecordpolicyFeatureGate.IsEnabled()
}
