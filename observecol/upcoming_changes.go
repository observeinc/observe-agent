package observecol

import (
	"context"
	_ "embed"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/otelcol"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

//go:embed upcoming_changes.yaml
var upcomingChangesYAML []byte

// UpcomingChange is a component whose configuration behaves differently, or whose output changes, in
// an upcoming Observe Agent release.
type UpcomingChange struct {
	// ID is the entry in upcoming_changes.yaml that matched.
	ID string
	// Component is the component's ID in the configuration, e.g. tail_sampling/observe.
	Component string
	Message   string
}

type upcomingChangeRule struct {
	ID       string    `yaml:"id"`
	Kind     string    `yaml:"kind"`
	Types    []string  `yaml:"types"`
	Upstream string    `yaml:"upstream"`
	RemoveIn string    `yaml:"remove_in"`
	Match    matchRule `yaml:"match"`
	Message  string    `yaml:"message"`
	Examples struct {
		Fires []map[string]any `yaml:"fires"`
		Quiet []map[string]any `yaml:"quiet"`
	} `yaml:"examples"`
}

type matchRule struct {
	Check string           `yaml:"check"`
	All   []matchCondition `yaml:"all"`
}

type matchCondition struct {
	Path        []string `yaml:"path"`
	Equals      any      `yaml:"equals"`
	Absent      bool     `yaml:"absent"`
	NotContains any      `yaml:"not_contains"`
}

// upcomingChangeChecks are matches upcoming_changes.yaml can't express as conditions.
var upcomingChangeChecks = map[string]func(cfg map[string]any) bool{
	"tail_sampling_invert_match": reliesOnInvertMatch,
	"hostmetrics_cpu_attribute":  dropsPerCPUSeries,
}

var (
	upcomingChangeKinds = []string{"receiver", "processor", "exporter", "connector", "extension"}
	versionPattern      = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)
	upcomingChangeRules = mustLoadUpcomingChangeRules(upcomingChangesYAML)
)

func mustLoadUpcomingChangeRules(content []byte) []upcomingChangeRule {
	rules, err := loadUpcomingChangeRules(content)
	if err != nil {
		panic(fmt.Sprintf("invalid upcoming_changes.yaml: %v", err))
	}
	return rules
}

func loadUpcomingChangeRules(content []byte) ([]upcomingChangeRule, error) {
	var file struct {
		Changes []upcomingChangeRule `yaml:"changes"`
	}
	if err := yaml.Unmarshal(content, &file); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, r := range file.Changes {
		if err := r.validate(); err != nil {
			return nil, fmt.Errorf("entry %q: %w", r.ID, err)
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("entry %q is defined twice", r.ID)
		}
		seen[r.ID] = true
	}
	return file.Changes, nil
}

func (r upcomingChangeRule) validate() error {
	switch {
	case r.ID == "":
		return fmt.Errorf("id is required")
	case !slices.Contains(upcomingChangeKinds, r.Kind):
		return fmt.Errorf("kind must be one of %v", upcomingChangeKinds)
	case len(r.Types) == 0:
		return fmt.Errorf("types is required")
	case !versionPattern.MatchString(r.Upstream), !versionPattern.MatchString(r.RemoveIn):
		return fmt.Errorf("upstream and remove_in must look like v1.2.3")
	case strings.TrimSpace(r.Message) == "":
		return fmt.Errorf("message is required")
	case len(r.Examples.Fires) == 0:
		return fmt.Errorf("examples.fires needs at least one configuration")
	case r.Match.Check != "" && len(r.Match.All) > 0:
		return fmt.Errorf("match takes either check or all, not both")
	}
	if _, ok := upcomingChangeChecks[r.Match.Check]; r.Match.Check != "" && !ok {
		return fmt.Errorf("unknown check %q", r.Match.Check)
	}
	for _, c := range r.Match.All {
		operators := 0
		for _, set := range []bool{c.Equals != nil, c.Absent, c.NotContains != nil} {
			if set {
				operators++
			}
		}
		if len(c.Path) == 0 || operators != 1 {
			return fmt.Errorf("each condition needs a path and exactly one of equals, absent or not_contains")
		}
	}
	return nil
}

func (r upcomingChangeRule) matches(cfg map[string]any) bool {
	if r.Match.Check != "" {
		return upcomingChangeChecks[r.Match.Check](cfg)
	}
	for _, c := range r.Match.All {
		if !c.holds(cfg) {
			return false
		}
	}
	return true
}

func (c matchCondition) holds(cfg map[string]any) bool {
	value, found := lookupPath(cfg, c.Path)
	switch {
	case c.Absent:
		return !found || value == nil
	case c.Equals != nil:
		return found && reflect.DeepEqual(value, c.Equals)
	default:
		list, _ := value.([]any)
		return !slices.ContainsFunc(list, func(v any) bool { return reflect.DeepEqual(v, c.NotContains) })
	}
}

func lookupPath(cfg map[string]any, path []string) (any, bool) {
	var current any = cfg
	for _, key := range path {
		m, ok := current.(map[string]any)
		if !ok {
			return nil, false
		}
		if current, ok = m[key]; !ok {
			return nil, false
		}
	}
	return current, true
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

// dropsPerCPUSeries reports whether a hostmetrics configuration has a CPU scraper with an enabled
// system.cpu.time or system.cpu.utilization metric that doesn't list the cpu attribute. Those
// metrics lose the cpu attribute from v0.157.0. The bundled configuration and the helm chart list
// it, so this never fires for them.
func dropsPerCPUSeries(cfg map[string]any) bool {
	scrapers, _ := cfg["scrapers"].(map[string]any)
	cpu, ok := scrapers["cpu"]
	if !ok {
		return false
	}
	cpuCfg, _ := cpu.(map[string]any)
	metrics, _ := cpuCfg["metrics"].(map[string]any)
	for _, m := range []struct {
		name             string
		enabledByDefault bool
	}{{"system.cpu.time", true}, {"system.cpu.utilization", false}} {
		metric, _ := metrics[m.name].(map[string]any)
		enabled := m.enabledByDefault
		if e, ok := metric["enabled"].(bool); ok {
			enabled = e
		}
		if !enabled {
			continue
		}
		attributes, _ := metric["attributes"].([]any)
		if !slices.Contains(attributes, any("cpu")) {
			return true
		}
	}
	return false
}

func matchUpcomingChanges(rules []upcomingChangeRule, conf map[string]any) []UpcomingChange {
	var changes []UpcomingChange
	for _, r := range rules {
		components, _ := conf[r.Kind+"s"].(map[string]any)
		for id, raw := range components {
			typ, _, _ := strings.Cut(id, "/")
			if !slices.Contains(r.Types, typ) {
				continue
			}
			cfg, _ := raw.(map[string]any)
			if r.matches(cfg) {
				changes = append(changes, UpcomingChange{ID: r.ID, Component: id, Message: strings.TrimSpace(r.Message)})
			}
		}
	}
	slices.SortFunc(changes, func(a, b UpcomingChange) int {
		return strings.Compare(a.Component+"\x00"+a.ID, b.Component+"\x00"+b.ID)
	})
	return changes
}

func upcomingChanges(conf map[string]any) []UpcomingChange {
	return matchUpcomingChanges(upcomingChangeRules, conf)
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
