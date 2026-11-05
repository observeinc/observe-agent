package observecol

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
)

const (
	tailSamplingType   = "tail_sampling"
	dropPolicyGuideURL = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/tailsamplingprocessor"

	invertMatchMessage = "Upcoming change: from OpenTelemetry Collector v0.152.0, which the next minor release of the Observe Agent includes, " +
		"invert_match policies in tail_sampling no longer exclude traces that other policies sample. " +
		"This processor has a top-level invert_match policy next to other policies, so it will sample more traces after that upgrade. " +
		"Replace its top-level invert_match policies with drop policies: " + dropPolicyGuideURL
)

// UpcomingChange is a component whose configuration behaves differently in an upcoming Observe Agent
// release.
type UpcomingChange struct {
	// Component is the component's ID in the configuration, e.g. tail_sampling/observe.
	Component string
	Message   string
}

// reliesOnInvertMatch reports whether a tail_sampling configuration behaves differently from
// v0.152.0: a top-level string_attribute, numeric_attribute or boolean_attribute policy with
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
			filter, _ := policy[typ].(map[string]any)
			if invert, _ := filter["invert_match"].(bool); invert {
				inverted++
			}
		}
	}
	return inverted > 0 && nonDrop > 1
}

func upcomingChanges(conf map[string]any) []UpcomingChange {
	processors, _ := conf["processors"].(map[string]any)
	var changes []UpcomingChange
	for id, raw := range processors {
		typ, _, _ := strings.Cut(id, "/")
		if typ != tailSamplingType {
			continue
		}
		cfg, _ := raw.(map[string]any)
		if reliesOnInvertMatch(cfg) {
			changes = append(changes, UpcomingChange{Component: id, Message: invertMatchMessage})
		}
	}
	slices.SortFunc(changes, func(a, b UpcomingChange) int { return strings.Compare(a.Component, b.Component) })
	return changes
}

type upcomingChangesConverter struct {
	logger *zap.Logger
}

func (c upcomingChangesConverter) Convert(_ context.Context, conf *confmap.Conf) error {
	for _, change := range upcomingChanges(conf.ToStringMap()) {
		c.logger.Warn(change.Message, zap.String("component", change.Component))
	}
	return nil
}

func newUpcomingChangesConverterFactory() confmap.ConverterFactory {
	return confmap.NewConverterFactory(func(set confmap.ConverterSettings) confmap.Converter {
		return upcomingChangesConverter{logger: set.Logger}
	})
}

// UpcomingChanges resolves the collector configuration and returns the components whose
// configuration behaves differently in an upcoming Observe Agent release.
func UpcomingChanges(ctx context.Context, settings *otelcol.CollectorSettings) ([]UpcomingChange, error) {
	resolverSettings := settings.ConfigProviderSettings.ResolverSettings
	resolverSettings.ConverterFactories = nil
	resolver, err := confmap.NewResolver(resolverSettings)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver for the upcoming changes check: %w", err)
	}
	defer func() { _ = resolver.Shutdown(ctx) }()
	conf, err := resolver.Resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve config for the upcoming changes check: %w", err)
	}
	return upcomingChanges(conf.ToStringMap()), nil
}
