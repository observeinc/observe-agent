// Package configconverter keeps user-authored OTel configuration working across
// renames of the agent's bundled components.
//
// The component IDs in our bundled config are the keys that otel_config_overrides
// and --config merge against, and confmap merges by literal key string. Renaming
// a bundled ID therefore does more than change a label: an override still naming
// the old ID stops merging into our component and instead defines a separate one
// holding only the override's own fields, which then fails validation. Note that
// this is independent of the upstream type aliases, which keep resolving the old
// spelling just fine.
//
// The converter registered here runs after the resolver has merged every config
// source and folds legacy IDs back into their canonical counterparts.
package configconverter

import (
	"context"
	"slices"
	"sort"
	"strings"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
)

// componentSections are the top-level keys whose immediate children are
// component IDs of the form "<type>" or "<type>/<name>".
var componentSections = map[string]struct{}{
	"receivers":  {},
	"processors": {},
	"exporters":  {},
	"connectors": {},
	"extensions": {},
}

// pipelineRefKeys are the keys within a single pipeline that hold a list of
// component IDs.
var pipelineRefKeys = map[string]struct{}{
	"receivers":  {},
	"processors": {},
	"exporters":  {},
}

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
	applied := make(map[string]string)
	if err := c.rewriteDefinitions(conf, applied); err != nil {
		return err
	}
	if err := c.rewriteReferences(conf, applied); err != nil {
		return err
	}
	warnApplied(ctx, applied)
	return nil
}

// warnApplied reports each remapped ID once. Nothing else will: the converter
// rewrites the ID before the collector ever sees it, so the upstream deprecated
// alias warning never fires for these.
func warnApplied(ctx context.Context, applied map[string]string) {
	if len(applied) == 0 {
		return
	}
	legacyIDs := make([]string, 0, len(applied))
	for legacy := range applied {
		legacyIDs = append(legacyIDs, legacy)
	}
	sort.Strings(legacyIDs)

	log := logger.FromCtx(ctx)
	for _, legacy := range legacyIDs {
		log.Warn(
			"remapped a deprecated component id in the otel configuration; update your configuration to use the new id",
			zap.String("deprecated_id", legacy),
			zap.String("new_id", applied[legacy]),
		)
	}
}

// rewriteDefinitions moves every leaf belonging to a legacy component ID under
// the canonical ID.
//
// Working leaf by leaf rather than moving whole component blocks is what makes a
// legacy block and an existing canonical block combine as a deep merge. The
// legacy leaves win on conflict, which is the intended precedence: the bundled
// config is canonical by construction, so anything found under a legacy ID was
// authored by the user.
func (c *converter) rewriteDefinitions(conf *confmap.Conf, applied map[string]string) error {
	rewritten := make(map[string]any)
	var stale []string

	for _, key := range conf.AllKeys() {
		section, id, rest, ok := splitComponentKey(key)
		if !ok {
			continue
		}
		canonical, found := c.mappings[id]
		if !found {
			continue
		}
		newKey := section + confmap.KeyDelimiter + canonical
		if rest != "" {
			newKey += confmap.KeyDelimiter + rest
		}
		rewritten[newKey] = conf.Get(key)
		stale = append(stale, key)
		applied[id] = canonical
	}

	if len(stale) == 0 {
		return nil
	}
	for _, key := range stale {
		conf.Delete(key)
	}
	return conf.Merge(confmap.NewFromStringMap(rewritten))
}

// rewriteReferences updates the ID lists that wire components into the service,
// so a pipeline still naming a legacy ID points at the component that
// rewriteDefinitions moved.
//
// AllKeys returns a snapshot rather than a live view, so rewriting keys while
// ranging over it is safe.
func (c *converter) rewriteReferences(conf *confmap.Conf, applied map[string]string) error {
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
func (c *converter) remapIDs(refs []any, applied map[string]string) []any {
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
		applied[id] = canonical
	}
	return out
}

// splitComponentKey breaks a flat key such as
// "exporters::otlphttp/observe::sending_queue::num_consumers" into its section,
// component ID and remainder. ok is false for keys that do not address a
// component definition.
func splitComponentKey(key string) (section, id, rest string, ok bool) {
	parts := strings.SplitN(key, confmap.KeyDelimiter, 3)
	if len(parts) < 2 {
		return "", "", "", false
	}
	if _, isSection := componentSections[parts[0]]; !isSection {
		return "", "", "", false
	}
	if len(parts) == 2 {
		return parts[0], parts[1], "", true
	}
	return parts[0], parts[1], parts[2], true
}

// isReferenceKey reports whether a flat key holds a list of component IDs.
func isReferenceKey(key string) bool {
	parts := strings.Split(key, confmap.KeyDelimiter)
	if len(parts) == 2 && parts[0] == "service" && parts[1] == "extensions" {
		return true
	}
	if len(parts) == 4 && parts[0] == "service" && parts[1] == "pipelines" {
		_, ok := pipelineRefKeys[parts[3]]
		return ok
	}
	return false
}
