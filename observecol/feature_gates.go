package observecol

import (
	"context"
	"errors"
	"maps"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/spf13/pflag"
	"go.opentelemetry.io/collector/featuregate"
	"go.uber.org/zap"
)

// Flag name and description copied directly from otel collector
const (
	featureGatesFlag            = "feature-gates"
	featureGatesFlagDescription = "Comma-delimited list of feature gate identifiers. Prefix with '-' to disable the feature. '+' or no prefix will enable the feature."
)

var featureGates []string

var internalFeatureFlagDefaults = map[string]bool{
	"exporter.prometheusremotewritexporter.EnableMultipleWorkers": true,
	"connector.spanmetrics.useSecondAsDefaultMetricsUnit":         false,
	"connector.spanmetrics.excludeResourceMetrics":                false,
	"connector.spanmetrics.includeCollectorInstanceID":            true,
	// Keep kubeletstats *.cpu.usage from the kubelet's UsageNanoCores, which Kubernetes Explorer
	// reads; the scrape-based calculation is missing on the first scrape.
	"receiver.kubeletstats.cpuUsageScrapeBased": false,
	// Emit only the apache receiver's new metric and attribute names. The receiver is new to the
	// agent, so starting on the new names means no user has to migrate off the old ones later.
	// disableOldFormatMetrics requires enableNewFormatMetrics; see internalFeatureFlagRequires.
	"receiver.apache.enableNewFormatMetrics":  true,
	"receiver.apache.disableOldFormatMetrics": true,
}

// internalFeatureFlagRequires maps a gate in internalFeatureFlagDefaults to a gate it requires.
// If the user explicitly disables the required gate, the dependent gate is disabled instead of
// getting its default, so a single opt-out flag doesn't leave an invalid pair. A dependent gate
// the user sets explicitly is always applied as given.
var internalFeatureFlagRequires = map[string]string{
	"receiver.apache.disableOldFormatMetrics": "receiver.apache.enableNewFormatMetrics",
}

func AddFeatureGateFlag(flags *pflag.FlagSet) {
	flags.StringSliceVar(&featureGates, featureGatesFlag, []string{}, featureGatesFlagDescription)
}

func ApplyFeatureGates(ctx context.Context) error {
	flags := make(map[string]bool)
	for _, f := range featureGates {
		if f[0] == '-' {
			flags[f[1:]] = false
		} else if f[0] == '+' {
			flags[f[1:]] = true
		} else {
			flags[f] = true
		}
	}

	// Apply internal defaults only if the user did not specify a value in the flag. A default
	// whose required gate the user explicitly disabled is replaced with false.
	userFlags := maps.Clone(flags)
	for id, enabled := range internalFeatureFlagDefaults {
		if _, ok := userFlags[id]; ok {
			continue
		}
		if required, ok := internalFeatureFlagRequires[id]; ok {
			if requiredEnabled, set := userFlags[required]; set && !requiredEnabled {
				enabled = false
			}
		}
		flags[id] = enabled
	}

	var errs error
	for id, enabled := range flags {
		err := featuregate.GlobalRegistry().Set(id, enabled)
		if err != nil {
			errs = errors.Join(errs, err)
		} else {
			logger.FromCtx(ctx).Debug("feature gate set", zap.String("id", id), zap.Bool("enabled", enabled))
		}
	}
	return errs
}
