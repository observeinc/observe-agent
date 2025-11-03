package observecol

import (
	"context"
	"errors"

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

	// Apply internal defaults only if the user did not specify a value in the flag.
	for id, enabled := range internalFeatureFlagDefaults {
		if _, ok := flags[id]; !ok {
			flags[id] = enabled
		}
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
