package observecol

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
)

const InvertMatchWarning = "tail_sampling policies with invert_match: true keep their behavior from before OpenTelemetry Collector v0.144.0: " +
	"a matching policy excludes the trace even when another policy samples it. " +
	"Upstream removed this behavior in v0.155.0, and a future Observe Agent release will remove it too. " +
	"Replace invert_match policies with drop policies: " +
	"https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/tailsamplingprocessor"

// invertMatchTailSamplingProcessors returns the IDs of the tail_sampling processors that set
// invert_match anywhere in their configuration, including in and, composite, and drop sub-policies.
func invertMatchTailSamplingProcessors(conf *confmap.Conf) []string {
	processors, _ := conf.ToStringMap()["processors"].(map[string]any)
	var ids []string
	for id, cfg := range processors {
		if typ, _, _ := strings.Cut(id, "/"); typ == "tail_sampling" && setsInvertMatch(cfg) {
			ids = append(ids, id)
		}
	}
	slices.Sort(ids)
	return ids
}

func setsInvertMatch(v any) bool {
	switch v := v.(type) {
	case map[string]any:
		for key, value := range v {
			if (key == "invert_match" && isTrue(value)) || setsInvertMatch(value) {
				return true
			}
		}
	case []any:
		return slices.ContainsFunc(v, setsInvertMatch)
	}
	return false
}

func isTrue(v any) bool {
	switch v := v.(type) {
	case bool:
		return v
	case string:
		b, err := strconv.ParseBool(v)
		return err == nil && b
	}
	return false
}

// InvertMatchTailSamplingProcessors resolves the collector configuration and returns the IDs of
// the tail_sampling processors that set invert_match.
func InvertMatchTailSamplingProcessors(ctx context.Context, settings *otelcol.CollectorSettings) ([]string, error) {
	resolver, err := confmap.NewResolver(settings.ConfigProviderSettings.ResolverSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create new resolver: %w", err)
	}
	conf, err := resolver.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while resolving config: %w", err)
	}
	return invertMatchTailSamplingProcessors(conf), nil
}

type invertMatchConverter struct {
	logger *zap.Logger
}

func (c invertMatchConverter) Convert(_ context.Context, conf *confmap.Conf) error {
	for _, id := range invertMatchTailSamplingProcessors(conf) {
		c.logger.Warn(InvertMatchWarning, zap.String("processor", id))
	}
	return nil
}

func newInvertMatchConverterFactory() confmap.ConverterFactory {
	return confmap.NewConverterFactory(func(set confmap.ConverterSettings) confmap.Converter {
		return invertMatchConverter{logger: set.Logger}
	})
}
