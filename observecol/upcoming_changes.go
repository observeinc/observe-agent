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
	hostMetricsType    = "hostmetrics"
	dropPolicyGuideURL = "https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/main/processor/tailsamplingprocessor"
	contribChangelog   = "https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/CHANGELOG.md"
	upcomingPrefix     = "Upcoming change in the next minor releases of the Observe Agent, which update the OpenTelemetry Collector from v0.151.0: "

	invertMatchMessage = "Upcoming change: from OpenTelemetry Collector v0.152.0, which the next minor release of the Observe Agent includes, " +
		"invert_match policies in tail_sampling no longer exclude traces that other policies sample. " +
		"This processor has a top-level invert_match policy next to other policies, so it will sample more traces after that upgrade. " +
		"Replace its top-level invert_match policies with drop policies: " + dropPolicyGuideURL

	httpCheckMessage = upcomingPrefix +
		"from v0.153.0, the httpcheck timing metrics (httpcheck.dns.lookup.duration, httpcheck.client.connection.duration, httpcheck.tls.handshake.duration, " +
		"httpcheck.client.request.duration, httpcheck.response.duration) report nanoseconds instead of milliseconds, so their values are 1,000,000 times larger. " +
		"Update dashboards and monitors that use them. Details: " + contribChangelog

	mongoDBMessage = upcomingPrefix +
		"the mongodb receiver requires MongoDB 4.4 or later (v0.155.0), its metric units follow UCUM (v0.157.0), " +
		"and mongodb.operation.latency.time becomes a cumulative sum instead of a gauge (v0.164.0). " +
		"Update dashboards and monitors that use these metrics. Details: " + contribChangelog

	sqlServerMessage = upcomingPrefix +
		"the sqlserver receiver's metric units follow UCUM, and sqlserver.lock.timeout.rate, if enabled, reports one data point per sqlserver.lock.timeout.type (v0.157.0); " +
		"server.address and server.port become resource attributes enabled by default and are removed from db.server.top_query log attributes (v0.161.0); " +
		"on direct database connections, counters from the Databases object carry sqlserver.database.name and the unlabeled _Total series is dropped, " +
		"so sqlserver.database.backup_or_restore.rate splits per database (v0.163.0). " +
		"Update dashboards and monitors that use these metrics. Details: " + contribChangelog

	postgreSQLMessage = upcomingPrefix +
		"from v0.158.0, the postgresql receiver collects postgresql.database.locks from every configured database instead of only the postgres database. " +
		"Update dashboards and monitors that use this metric. Details: " + contribChangelog

	hostMetricsCPUMessage = upcomingPrefix +
		"from v0.157.0, the hostmetrics CPU scraper reports system.cpu.time and system.cpu.utilization without the cpu attribute unless it is listed, " +
		"so per-CPU series are combined. To keep them, set attributes: [cpu, state] on system.cpu.time, and on system.cpu.utilization if it is enabled. " +
		"Details: " + contribChangelog
)

// receiverMessages are changes that affect every receiver of a type. None of these types are in
// the agent's bundled configuration or the helm chart, so a receiver of these types is user config.
var receiverMessages = map[string]string{
	"http_check": httpCheckMessage,
	"httpcheck":  httpCheckMessage,
	"mongodb":    mongoDBMessage,
	"sqlserver":  sqlServerMessage,
	"postgresql": postgreSQLMessage,
}

// UpcomingChange is a component whose configuration behaves differently, or whose output changes, in
// an upcoming Observe Agent release.
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

func upcomingChanges(conf map[string]any) []UpcomingChange {
	var changes []UpcomingChange
	processors, _ := conf["processors"].(map[string]any)
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
	receivers, _ := conf["receivers"].(map[string]any)
	for id, raw := range receivers {
		typ, _, _ := strings.Cut(id, "/")
		if message, ok := receiverMessages[typ]; ok {
			changes = append(changes, UpcomingChange{Component: id, Message: message})
			continue
		}
		cfg, _ := raw.(map[string]any)
		if typ == hostMetricsType && dropsPerCPUSeries(cfg) {
			changes = append(changes, UpcomingChange{Component: id, Message: hostMetricsCPUMessage})
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
