package configconverter

// LegacyComponentIDs maps a component ID that appeared in a previous release's
// bundled configuration to the ID that replaces it.
//
// Entries are keyed on the full component ID ("otlphttp/observe") rather than
// the bare type ("otlphttp"). The breakage this table repairs is specific to IDs
// that collide with our bundled config; a user's own component that happens to
// use a legacy-style type name is not affected by our renames, because the
// upstream type alias still resolves it. Keying on the full ID leaves those
// alone.
//
// Add an entry whenever a bundled component ID is renamed, and keep it for at
// least as long as upstream keeps its own type aliases. Removing an entry is
// itself a breaking change. File-storage clients name files from the component
// type and name, so a rename also changes the on-disk queue/checkpoint
// filename; MigrateFileStorage uses this table to rename those files.
var LegacyComponentIDs = map[string]string{
	// Renamed to follow the upstream lower_snake_case component naming
	// convention (open-telemetry/opentelemetry-collector#14208).
	"otlphttp/observe":                    "otlp_http/observe",
	"otlphttp/observemetrics":             "otlp_http/observemetrics",
	"otlphttp/observetracing":             "otlp_http/observetracing",
	"otlphttp/agentheartbeat":             "otlp_http/agentheartbeat",
	"filelog/host_monitoring":             "file_log/host_monitoring",
	"filestats/agent":                     "file_stats/agent",
	"hostmetrics/host-monitoring-host":    "host_metrics/host-monitoring-host",
	"hostmetrics/host-monitoring-process": "host_metrics/host-monitoring-process",
}
