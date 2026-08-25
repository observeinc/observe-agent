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
// Add an entry in the same PR that renames a bundled component ID, and keep it
// for at least as long as upstream keeps its own type aliases. Removing an entry
// is itself a breaking change.
var LegacyComponentIDs = map[string]string{}
