package observecol

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	tailsamplinglegacyprocessor "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	metricnoop "go.opentelemetry.io/otel/metric/noop"
	tracenoop "go.opentelemetry.io/otel/trace/noop"
	"go.uber.org/zap"
)

const (
	tailSamplingType       = "tail_sampling"
	tailSamplingLegacyType = "tailsampling_legacy"
	dropPolicyGuideURL     = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/tailsamplingprocessor"
)

// InvertMatchSwap is a tail_sampling processor whose configuration relies on invert_match
// excluding traces, the behavior OpenTelemetry Collector v0.155.0 removed.
type InvertMatchSwap struct {
	// Processor is the processor's ID in the configuration, e.g. tail_sampling/observe.
	Processor string
	// LegacyID is the ID the processor runs under when swapped, e.g. tailsampling_legacy/observe.
	LegacyID string
	// Err is set when tailsampling_legacy can't run the configuration, so the processor isn't swapped.
	Err error
}

func (s InvertMatchSwap) Message() string {
	if s.Err != nil {
		return "This tail_sampling processor relies on invert_match policies excluding traces, which OpenTelemetry Collector v0.155.0 removed, " +
			"but its configuration can't run on " + tailSamplingLegacyType + " (" + s.Err.Error() + "). " +
			"It runs on the current tail_sampling processor, where those invert_match policies no longer exclude traces. " +
			"Replace them with drop policies: " + dropPolicyGuideURL
	}
	return "This tail_sampling processor relies on invert_match policies excluding traces, which OpenTelemetry Collector v0.155.0 removed, " +
		"so the Observe Agent runs it as " + s.LegacyID + " to keep that behavior. " +
		"Replace its top-level invert_match policies with drop policies to move to the current tail_sampling processor: " + dropPolicyGuideURL
}

// reliesOnInvertMatch reports whether a tail_sampling configuration behaves differently since
// v0.155.0: a top-level string_attribute, numeric_attribute or boolean_attribute policy with
// invert_match, next to at least one other top-level policy that isn't a drop policy. invert_match
// inside and, composite, drop and not policies, and a lone inverted policy, behave the same.
func reliesOnInvertMatch(cfg map[string]any) bool {
	policies, _ := cfg["policies"].([]any)
	inverted, nonDrop := 0, 0
	for _, p := range policies {
		policy, _ := p.(map[string]any)
		typ, _ := policy["type"].(string)
		if typ == "drop" {
			continue
		}
		nonDrop++
		switch typ {
		case "string_attribute", "numeric_attribute", "boolean_attribute":
			if filter, _ := policy[typ].(map[string]any); isTrue(filter["invert_match"]) {
				inverted++
			}
		}
	}
	return inverted > 0 && nonDrop > 1
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

// legacyConfigError reports why tailsampling_legacy can't run a tail_sampling configuration, such as
// settings added after v0.151.0.
func legacyConfigError(cfg map[string]any) error {
	factory := tailsamplinglegacyprocessor.NewFactory()
	legacyCfg := factory.CreateDefaultConfig()
	if err := confmap.NewFromStringMap(cfg).Unmarshal(legacyCfg); err != nil {
		return err
	}
	if err := confmap.Validate(legacyCfg); err != nil {
		return err
	}
	return tailsamplinglegacyprocessor.ValidatePolicies(legacyCfg, component.TelemetrySettings{
		Logger:         zap.NewNop(),
		MeterProvider:  metricnoop.NewMeterProvider(),
		TracerProvider: tracenoop.NewTracerProvider(),
	})
}

func planInvertMatchSwaps(conf map[string]any) []InvertMatchSwap {
	processors, _ := conf["processors"].(map[string]any)
	var swaps []InvertMatchSwap
	for id, raw := range processors {
		typ, name, hasName := strings.Cut(id, "/")
		if typ != tailSamplingType {
			continue
		}
		cfg, _ := raw.(map[string]any)
		if !reliesOnInvertMatch(cfg) {
			continue
		}
		swap := InvertMatchSwap{Processor: id, LegacyID: tailSamplingLegacyType}
		if hasName {
			swap.LegacyID += "/" + name
		}
		if _, exists := processors[swap.LegacyID]; exists {
			swap.Err = fmt.Errorf("processor %s is already defined", swap.LegacyID)
		} else {
			swap.Err = legacyConfigError(cfg)
		}
		swaps = append(swaps, swap)
	}
	slices.SortFunc(swaps, func(a, b InvertMatchSwap) int { return strings.Compare(a.Processor, b.Processor) })
	return swaps
}

func applyInvertMatchSwaps(conf map[string]any, swaps []InvertMatchSwap) {
	processors, _ := conf["processors"].(map[string]any)
	renamed := map[string]string{}
	for _, s := range swaps {
		if s.Err != nil {
			continue
		}
		processors[s.LegacyID] = processors[s.Processor]
		delete(processors, s.Processor)
		renamed[s.Processor] = s.LegacyID
	}
	service, _ := conf["service"].(map[string]any)
	pipelines, _ := service["pipelines"].(map[string]any)
	for _, p := range pipelines {
		pipeline, _ := p.(map[string]any)
		ids, _ := pipeline["processors"].([]any)
		for i, id := range ids {
			if to, ok := renamed[fmt.Sprint(id)]; ok {
				ids[i] = to
			}
		}
	}
}

type invertMatchConverter struct {
	logger *zap.Logger
}

func (c invertMatchConverter) Convert(_ context.Context, conf *confmap.Conf) error {
	m := conf.ToStringMap()
	swaps := planInvertMatchSwaps(m)
	if len(swaps) == 0 {
		return nil
	}
	for _, s := range swaps {
		c.logger.Warn(s.Message(), zap.String("processor", s.Processor))
	}
	applyInvertMatchSwaps(m, swaps)
	*conf = *confmap.NewFromStringMap(m)
	return nil
}

func newInvertMatchConverterFactory() confmap.ConverterFactory {
	return confmap.NewConverterFactory(func(set confmap.ConverterSettings) confmap.Converter {
		return invertMatchConverter{logger: set.Logger}
	})
}

// InvertMatchSwaps resolves the collector configuration as written, before the agent swaps any
// processor to tailsampling_legacy, and returns the tail_sampling processors that rely on
// invert_match excluding traces.
func InvertMatchSwaps(ctx context.Context, settings *otelcol.CollectorSettings) ([]InvertMatchSwap, error) {
	resolverSettings := settings.ConfigProviderSettings.ResolverSettings
	resolverSettings.ConverterFactories = nil
	resolver, err := confmap.NewResolver(resolverSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver for the invert_match check: %w", err)
	}
	defer func() { _ = resolver.Shutdown(ctx) }()
	conf, err := resolver.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config for the invert_match check: %w", err)
	}
	return planInvertMatchSwaps(conf.ToStringMap()), nil
}
