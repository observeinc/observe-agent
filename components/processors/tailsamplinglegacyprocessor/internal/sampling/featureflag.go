// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package sampling // import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/sampling"

import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/metadata"

func IsInvertDecisionsDisabled() bool {
	return metadata.ProcessorTailsamplingprocessorDisableinvertdecisionsFeatureGate.IsEnabled()
}
