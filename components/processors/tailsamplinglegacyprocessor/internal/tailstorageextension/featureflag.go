// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package tailstorageextension // import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/tailstorageextension"

import (
	"github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/metadata"
)

var FeatureGateID = metadata.ProcessorTailsamplingprocessorTailstorageextensionFeatureGate.ID()

func IsFeatureGateEnabled() bool {
	return metadata.ProcessorTailsamplingprocessorTailstorageextensionFeatureGate.IsEnabled()
}
