// Package configconverter keeps user-authored OTel configuration working across
// renames of the agent's bundled components.
//
// The component IDs in our bundled config are the keys that otel_config_overrides
// and --config merge against, and confmap merges by literal key string. Renaming
// a bundled ID therefore does more than change a label: an override still naming
// the old ID stops merging into our component and instead defines a separate one
// holding only the override's own fields, which then fails validation. This is
// independent of the upstream type aliases, which still resolve the old spelling.
//
// The converter runs after the resolver has merged every config source and folds
// legacy IDs back into their canonical counterparts.
package configconverter

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
)

// componentSections are the top-level keys whose immediate children are
// component IDs of the form "<type>" or "<type>/<name>".
var componentSections = []string{"receivers", "processors", "exporters", "connectors", "extensions"}

// pipelineRefKeys are the keys within a single pipeline that hold a list of
// component IDs.
var pipelineRefKeys = []string{"receivers", "processors", "exporters"}

type converter struct {
	mappings map[string]string
}

// NewFactory returns a converter factory that applies LegacyComponentIDs.
func NewFactory() confmap.ConverterFactory {
	return newFactory(LegacyComponentIDs)
}

func newFactory(mappings map[string]string) confmap.ConverterFactory {
	return confmap.NewConverterFactory(func(confmap.ConverterSettings) confmap.Converter {
		return &converter{mappings: mappings}
	})
}

func (c *converter) Convert(ctx context.Context, conf *confmap.Conf) error {
	if conf == nil || len(c.mappings) == 0 {
		return nil
	}
	// The set of legacy IDs actually encountered, for the warning below.
	applied := make(map[string]struct{})
	if err := c.rewriteDefinitions(conf, applied); err != nil {
		return err
	}
	if err := c.rewriteReferences(conf, applied); err != nil {
		return err
	}
	c.warnApplied(ctx, applied)
	return nil
}

// warnApplied reports each remapped ID once. Nothing else reports it: the
// converter rewrites the ID before the collector sees it, so the upstream
// deprecated alias warning never fires for these.
func (c *converter) warnApplied(ctx context.Context, applied map[string]struct{}) {
	// logger.FromCtx builds a fresh logger when the context carries none, so
	// stay out of it on the common path where nothing was remapped.
	if len(applied) == 0 {
		return
	}
	log := logger.FromCtx(ctx)
	for _, legacy := range slices.Sorted(maps.Keys(applied)) {
		log.Warn(
			"remapped a deprecated component id in the otel configuration; update your configuration to use the new id",
			zap.String("deprecated_id", legacy),
			zap.String("new_id", c.mappings[legacy]),
		)
	}
}

// rewriteDefinitions moves every leaf belonging to a legacy component ID under
// the canonical ID.
//
// Rewriting individual leaves rather than whole blocks is what lets a legacy
// block deep-merge with an existing canonical one. Legacy leaves win on
// conflict, which is the intended precedence: the bundled config is canonical by
// construction, so anything under a legacy ID was authored by the user.
func (c *converter) rewriteDefinitions(conf *confmap.Conf, applied map[string]struct{}) error {
	rewritten := make(map[string]any)
	var stale []string

	for _, key := range conf.AllKeys() {
		newKey, legacy, ok := c.remapDefinitionKey(key)
		if !ok {
			continue
		}
		rewritten[newKey] = conf.Get(key)
		stale = append(stale, key)
		applied[legacy] = struct{}{}
	}

	if len(stale) == 0 {
		return nil
	}
	// Deleting re-flattens the whole map, so do it once the walk is done.
	for _, key := range stale {
		conf.Delete(key)
	}
	return conf.Merge(confmap.NewFromStringMap(rewritten))
}

// remapDefinitionKey swaps the component-ID segment of a flat definition key
// such as "exporters::otlphttp/observe::sending_queue::num_consumers",
// returning the new key and the legacy ID it replaced. ok is false unless the
// key addresses a component definition whose ID is in the table.
func (c *converter) remapDefinitionKey(key string) (newKey, legacy string, ok bool) {
	// A cap of 3 leaves any remainder untouched in the final part, so joining
	// back together reproduces the key with only the ID changed.
	parts := strings.SplitN(key, confmap.KeyDelimiter, 3)
	if len(parts) < 2 || !slices.Contains(componentSections, parts[0]) {
		return "", "", false
	}
	canonical, found := c.mappings[parts[1]]
	if !found {
		return "", "", false
	}
	legacy, parts[1] = parts[1], canonical
	return strings.Join(parts, confmap.KeyDelimiter), legacy, true
}

// rewriteReferences updates the ID lists that wire components into the service,
// so a pipeline still naming a legacy ID points at the component that
// rewriteDefinitions moved.
//
// AllKeys returns a snapshot rather than a live view, so rewriting keys while
// ranging over it is safe.
func (c *converter) rewriteReferences(conf *confmap.Conf, applied map[string]struct{}) error {
	for _, key := range conf.AllKeys() {
		if !isReferenceKey(key) {
			continue
		}
		refs, ok := conf.Get(key).([]any)
		if !ok {
			continue
		}
		remapped := c.remapIDs(refs, applied)
		if remapped == nil {
			continue
		}
		// Delete before merging: with the confmap.enableMergeAppendOption
		// feature gate enabled, merging a list appends to the existing one
		// rather than replacing it.
		conf.Delete(key)
		if err := conf.Merge(confmap.NewFromStringMap(map[string]any{key: remapped})); err != nil {
			return err
		}
	}
	return nil
}

// remapIDs returns refs with every legacy ID replaced by its canonical form, or
// nil when the list holds none. A non-string entry never matches: the failed
// assertion yields "", and no component ID is empty.
func (c *converter) remapIDs(refs []any, applied map[string]struct{}) []any {
	var out []any
	for i, ref := range refs {
		id, _ := ref.(string)
		canonical, found := c.mappings[id]
		if !found {
			continue
		}
		if out == nil {
			out = slices.Clone(refs)
		}
		out[i] = canonical
		applied[id] = struct{}{}
	}
	return out
}

// isReferenceKey reports whether a flat key holds a list of component IDs,
// namely "service::extensions" or
// "service::pipelines::<pipeline>::<receivers|processors|exporters>".
func isReferenceKey(key string) bool {
	parts := strings.Split(key, confmap.KeyDelimiter)
	if len(parts) < 2 || parts[0] != "service" {
		return false
	}
	switch len(parts) {
	case 2:
		return parts[1] == "extensions"
	case 4:
		return parts[1] == "pipelines" && slices.Contains(pipelineRefKeys, parts[3])
	}
	return false
}
