package config

import (
	"bytes"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/xconfmap"
	"gopkg.in/yaml.v3"
)

// ComponentKind is a top-level section of an OpenTelemetry Collector configuration whose keys are
// component IDs.
type ComponentKind string

const (
	Receivers  ComponentKind = "receivers"
	Processors ComponentKind = "processors"
	Exporters  ComponentKind = "exporters"
	Connectors ComponentKind = "connectors"
	Extensions ComponentKind = "extensions"
)

// ComponentKinds lists every component kind in the order the collector documents them.
var ComponentKinds = []ComponentKind{Receivers, Processors, Exporters, Connectors, Extensions}

// pipelineRefKeys are the keys within a pipeline that hold component IDs.
var pipelineRefKeys = []string{"receivers", "processors", "exporters"}

// ComponentID is a component ID of the form "<type>" or "<type>/<name>".
type ComponentID struct {
	Type string
	Name string
}

// ParseComponentID splits an ID such as "otlp_http/observe" into its type and name.
func ParseComponentID(id string) ComponentID {
	typ, name, _ := strings.Cut(id, "/")
	return ComponentID{Type: typ, Name: name}
}

func (id ComponentID) String() string {
	if id.Name == "" {
		return id.Type
	}
	return id.Type + "/" + id.Name
}

// OtelConfig is an editable OpenTelemetry Collector configuration.
//
// Values are held as the raw nested map that confmap uses, so a config loaded with
// NewOtelConfigFromConfmap keeps the ExpandedValue metadata from environment expansion when it is
// written back with ApplyTo.
type OtelConfig struct {
	raw map[string]any
}

// NewOtelConfigFromYAML parses a collector configuration.
func NewOtelConfigFromYAML(data []byte) (*OtelConfig, error) {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing otel config: %w", err)
	}
	if raw == nil {
		raw = map[string]any{}
	}
	return &OtelConfig{raw: raw}, nil
}

// NewOtelConfigFromConfmap copies a resolved collector configuration.
func NewOtelConfigFromConfmap(conf *confmap.Conf) *OtelConfig {
	raw := xconfmap.ToStringMapRaw(conf)
	if raw == nil {
		raw = map[string]any{}
	}
	return &OtelConfig{raw: raw}
}

// ApplyTo replaces the content of conf with this configuration.
func (c *OtelConfig) ApplyTo(conf *confmap.Conf) error {
	c.dropEmptySections()
	for key := range xconfmap.ToStringMapRaw(conf) {
		conf.Delete(key)
	}
	return conf.Merge(confmap.NewFromStringMap(c.raw))
}

// ToYAML renders the configuration with a 2-space indent. Empty component sections are left out;
// the remove methods also delete the sections they leave empty.
func (c *OtelConfig) ToYAML() (string, error) {
	c.dropEmptySections()
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(c.raw); err != nil {
		return "", fmt.Errorf("encoding otel config: %w", err)
	}
	if err := enc.Close(); err != nil {
		return "", fmt.Errorf("encoding otel config: %w", err)
	}
	return buf.String(), nil
}

// Pipelines returns the sorted pipeline IDs.
func (c *OtelConfig) Pipelines() []string {
	return slices.Sorted(maps.Keys(c.pipelines()))
}

// Components returns the sorted IDs of the components of a kind.
func (c *OtelConfig) Components(kind ComponentKind) []string {
	section, _ := c.raw[string(kind)].(map[string]any)
	return slices.Sorted(maps.Keys(section))
}

// Component returns the component with the given ID, or nil if it is not defined.
func (c *OtelConfig) Component(kind ComponentKind, id string) *Component {
	section, _ := c.raw[string(kind)].(map[string]any)
	if _, ok := section[id]; !ok {
		return nil
	}
	return &Component{cfg: c, Kind: kind, ID: id}
}

// ComponentsOfType returns the components of a kind with the given type, for example every
// "otlp_http" exporter, sorted by ID.
func (c *OtelConfig) ComponentsOfType(kind ComponentKind, typ string) []*Component {
	var out []*Component
	for _, id := range c.Components(kind) {
		if ParseComponentID(id).Type == typ {
			out = append(out, &Component{cfg: c, Kind: kind, ID: id})
		}
	}
	return out
}

// RemovePipeline deletes a pipeline. It reports whether the pipeline existed.
func (c *OtelConfig) RemovePipeline(name string) bool {
	pipelines := c.pipelines()
	if _, ok := pipelines[name]; !ok {
		return false
	}
	delete(pipelines, name)
	return true
}

// RetainPipelines deletes every pipeline that is not listed. It fails if a listed pipeline is not
// defined.
func (c *OtelConfig) RetainPipelines(names ...string) error {
	pipelines := c.pipelines()
	for _, name := range names {
		if _, ok := pipelines[name]; !ok {
			return fmt.Errorf("pipeline %q is not defined", name)
		}
	}
	for name := range pipelines {
		if !slices.Contains(names, name) {
			delete(pipelines, name)
		}
	}
	return nil
}

// RetainExtensions deletes every extension that is not listed, and sets service.extensions to the
// list. It fails if a listed extension is not defined.
func (c *OtelConfig) RetainExtensions(ids ...string) error {
	section, _ := c.raw[string(Extensions)].(map[string]any)
	for _, id := range ids {
		if _, ok := section[id]; !ok {
			return fmt.Errorf("extension %q is not defined", id)
		}
	}
	for id := range section {
		if !slices.Contains(ids, id) {
			delete(section, id)
		}
	}
	if service := c.service(); service != nil {
		if len(ids) == 0 {
			delete(service, "extensions")
		} else {
			service["extensions"] = toAnyList(ids)
		}
	}
	c.dropEmptySections()
	return nil
}

// RemoveComponent deletes a component and every reference to it in the service. It reports whether
// the component was defined.
func (c *OtelConfig) RemoveComponent(kind ComponentKind, id string) bool {
	section, _ := c.raw[string(kind)].(map[string]any)
	_, found := section[id]
	delete(section, id)
	c.editRefs(kind, func(ref string) (string, bool) { return ref, ref != id })
	c.dropEmptySections()
	return found
}

// RemoveUnusedComponents deletes every component that no pipeline references, and every extension
// that service.extensions doesn't list.
//
// A connector is only usable if one pipeline exports to it and another receives from it. Other
// connectors are also removed from the pipelines that list them; this repeats until no connector
// changes, as one removal can make another connector unusable.
func (c *OtelConfig) RemoveUnusedComponents() {
	for {
		exported, received := map[string]bool{}, map[string]bool{}
		for _, p := range c.pipelines() {
			pipeline, _ := p.(map[string]any)
			for _, id := range refList(pipeline["exporters"]) {
				exported[id] = true
			}
			for _, id := range refList(pipeline["receivers"]) {
				received[id] = true
			}
		}
		changed := false
		for _, id := range c.Components(Connectors) {
			if !exported[id] || !received[id] {
				c.RemoveComponent(Connectors, id)
				changed = true
			}
		}
		if !changed {
			break
		}
	}

	for _, kind := range ComponentKinds {
		used := map[string]bool{}
		for _, id := range c.references(kind) {
			used[id] = true
		}
		section, _ := c.raw[string(kind)].(map[string]any)
		for id := range section {
			if !used[id] {
				delete(section, id)
			}
		}
	}
	c.dropEmptySections()
}

// RenameComponent changes the ID of a component and every reference to it in the service. If a
// component with the new ID is already defined, the two are merged and the values of the renamed
// component win on conflict. It fails if no component has the old ID.
func (c *OtelConfig) RenameComponent(kind ComponentKind, from, to ComponentID) error {
	oldID, newID := from.String(), to.String()
	section, _ := c.raw[string(kind)].(map[string]any)
	value, ok := section[oldID]
	if !ok {
		return fmt.Errorf("%s %q is not defined", kind.singular(), oldID)
	}
	if oldID == newID {
		return nil
	}
	delete(section, oldID)
	if existing, ok := section[newID]; ok {
		value = deepMerge(existing, value)
	}
	section[newID] = value

	c.editRefs(kind, func(ref string) (string, bool) {
		if ref == oldID {
			return newID, true
		}
		return ref, true
	})
	return nil
}

// Validate checks that every service reference is a component ID of a defined component, and that
// every pipeline has at least one receiver and one exporter.
func (c *OtelConfig) Validate() error {
	var errs []error
	for _, name := range c.Pipelines() {
		pipeline, _ := c.pipelines()[name].(map[string]any)
		for _, key := range pipelineRefKeys {
			problems := refProblems(pipeline[key])
			for _, p := range problems {
				errs = append(errs, fmt.Errorf("pipeline %q %s: %s", name, key, p))
			}
			refs := refList(pipeline[key])
			if key != "processors" && len(refs) == 0 && len(problems) == 0 {
				errs = append(errs, fmt.Errorf("pipeline %q has no %s", name, key))
			}
			for _, id := range refs {
				if !c.isDefined(ComponentKind(key), id) && (key == "processors" || !c.isDefined(Connectors, id)) {
					errs = append(errs, fmt.Errorf("pipeline %q references %s %q, which is not defined", name, ComponentKind(key).singular(), id))
				}
			}
		}
	}
	if service := c.service(); service != nil {
		for _, p := range refProblems(service["extensions"]) {
			errs = append(errs, fmt.Errorf("service extensions: %s", p))
		}
		for _, id := range refList(service["extensions"]) {
			if !c.isDefined(Extensions, id) {
				errs = append(errs, fmt.Errorf("service references extension %q, which is not defined", id))
			}
		}
	}
	return errors.Join(errs...)
}

// Component is a reference to one component definition in an OtelConfig.
type Component struct {
	cfg  *OtelConfig
	Kind ComponentKind
	ID   string
}

// Get returns the value at a path in the component's configuration, for example
// Get("sending_queue", "enabled"). A path segment selects a list element by its index.
func (comp *Component) Get(path ...string) (any, bool) {
	return getPath(comp.root(false), path)
}

// Set writes a value at a path in the component's configuration and creates the maps it needs. It
// fails if the path goes through a scalar or a list index that is out of range.
func (comp *Component) Set(value any, path ...string) error {
	if err := setPath(comp.root(true), value, path); err != nil {
		return fmt.Errorf("%s %q: %w", comp.Kind.singular(), comp.ID, err)
	}
	return nil
}

// Delete removes the value at a path in the component's configuration. A list element is removed
// from its list. It reports whether the value existed.
func (comp *Component) Delete(path ...string) bool {
	return deletePath(comp.root(false), path)
}

// root returns the component's configuration map. With create, it creates the section and the map
// if they are missing; a component defined as null gets an empty map.
func (comp *Component) root(create bool) map[string]any {
	section, _ := comp.cfg.raw[string(comp.Kind)].(map[string]any)
	if section == nil && create {
		section = map[string]any{}
		comp.cfg.raw[string(comp.Kind)] = section
	}
	m, _ := section[comp.ID].(map[string]any)
	if m == nil && create {
		m = map[string]any{}
		section[comp.ID] = m
	}
	return m
}

// Telemetry returns the collector's own telemetry settings, service.telemetry.
func (c *OtelConfig) Telemetry() *Telemetry {
	return &Telemetry{cfg: c}
}

// Telemetry is a reference to service.telemetry in an OtelConfig.
type Telemetry struct {
	cfg *OtelConfig
}

// Get returns the value at a path in service.telemetry, for example Get("metrics", "readers", "0").
// A path segment selects a list element by its index.
func (t *Telemetry) Get(path ...string) (any, bool) {
	return getPath(t.root(false), path)
}

// Set writes a value at a path in service.telemetry and creates the maps it needs. It fails if the
// path goes through a scalar or a list index that is out of range.
func (t *Telemetry) Set(value any, path ...string) error {
	if err := setPath(t.root(true), value, path); err != nil {
		return fmt.Errorf("service.telemetry: %w", err)
	}
	return nil
}

// Delete removes the value at a path in service.telemetry. A list element is removed from its list.
// It reports whether the value existed.
func (t *Telemetry) Delete(path ...string) bool {
	return deletePath(t.root(false), path)
}

func (t *Telemetry) root(create bool) map[string]any {
	service := t.cfg.service()
	if service == nil && create {
		service = map[string]any{}
		t.cfg.raw["service"] = service
	}
	m, _ := service["telemetry"].(map[string]any)
	if m == nil && create {
		m = map[string]any{}
		service["telemetry"] = m
	}
	return m
}

// getPath walks maps by key and lists by index.
func getPath(root map[string]any, path []string) (any, bool) {
	if root == nil {
		return nil, false
	}
	var cur any = root
	for _, key := range path {
		switch c := cur.(type) {
		case map[string]any:
			v, ok := c[key]
			if !ok {
				return nil, false
			}
			cur = v
		case []any:
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(c) {
				return nil, false
			}
			cur = c[idx]
		default:
			return nil, false
		}
	}
	return cur, true
}

// setPath writes value at path below root. Missing or null map entries become maps; list elements
// must already exist.
func setPath(root map[string]any, value any, path []string) error {
	if len(path) == 0 {
		return errors.New("path is empty")
	}
	var cur any = root
	for i, key := range path {
		last := i == len(path)-1
		switch c := cur.(type) {
		case map[string]any:
			if last {
				c[key] = value
				return nil
			}
			if c[key] == nil {
				c[key] = map[string]any{}
			}
			cur = c[key]
		case []any:
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(c) {
				return fmt.Errorf("%s: index %q is out of range", strings.Join(path[:i+1], "."), key)
			}
			if last {
				c[idx] = value
				return nil
			}
			if c[idx] == nil {
				c[idx] = map[string]any{}
			}
			cur = c[idx]
		default:
			return fmt.Errorf("%s is not a map or a list", strings.Join(path[:i], "."))
		}
	}
	return nil
}

// deletePath removes the value at path below root.
func deletePath(root map[string]any, path []string) bool {
	if len(path) == 0 {
		return false
	}
	parentPath, key := path[:len(path)-1], path[len(path)-1]
	parent, ok := getPath(root, parentPath)
	if !ok {
		return false
	}
	switch p := parent.(type) {
	case map[string]any:
		if _, ok := p[key]; !ok {
			return false
		}
		delete(p, key)
		return true
	case []any:
		idx, err := strconv.Atoi(key)
		if err != nil || idx < 0 || idx >= len(p) {
			return false
		}
		// The shorter list replaces the old one in its parent.
		return setPath(root, slices.Delete(slices.Clone(p), idx, idx+1), parentPath) == nil
	}
	return false
}

func (k ComponentKind) singular() string {
	return strings.TrimSuffix(string(k), "s")
}

func (c *OtelConfig) service() map[string]any {
	service, _ := c.raw["service"].(map[string]any)
	return service
}

func (c *OtelConfig) pipelines() map[string]any {
	pipelines, _ := c.service()["pipelines"].(map[string]any)
	return pipelines
}

func (c *OtelConfig) isDefined(kind ComponentKind, id string) bool {
	section, _ := c.raw[string(kind)].(map[string]any)
	_, ok := section[id]
	return ok
}

// refKeys returns the pipeline keys that can reference a component of the kind. Connectors are
// listed as both receivers and exporters.
func refKeys(kind ComponentKind) []string {
	switch kind {
	case Connectors:
		return []string{"receivers", "exporters"}
	case Receivers, Processors, Exporters:
		return []string{string(kind)}
	}
	return nil
}

// references returns every ID in the service that can reference a component of the kind.
func (c *OtelConfig) references(kind ComponentKind) []string {
	var out []string
	if kind == Extensions {
		return refList(c.service()["extensions"])
	}
	for _, p := range c.pipelines() {
		pipeline, _ := p.(map[string]any)
		for _, key := range refKeys(kind) {
			out = append(out, refList(pipeline[key])...)
		}
	}
	return out
}

// editRefs rewrites every reference list in the service that can hold a component of the kind. For
// each ID, edit returns the new ID and whether to keep the reference; a list that then holds an ID
// twice keeps the first one. Items that the edit doesn't change keep their original form, such as
// an xconfmap.ExpandedValue, and lists with no change are not rewritten.
func (c *OtelConfig) editRefs(kind ComponentKind, edit func(id string) (string, bool)) {
	service := c.service()
	if service == nil {
		return
	}
	apply := func(m map[string]any, key string) {
		v, ok := m[key]
		if !ok {
			return
		}
		items, isList := v.([]any)
		if !isList {
			items = toAnyList(refList(v))
		}
		if expanded, ok := v.(xconfmap.ExpandedValue); ok {
			// A whole value set from a ${...} reference; a changed list replaces it.
			items, isList = expanded.Value.([]any)
			if !isList {
				items = toAnyList(refList(expanded.Value))
			}
		}
		out := make([]any, 0, len(items))
		seen := map[string]bool{}
		changed := false
		for _, item := range items {
			id, ok := refID(item)
			if !ok {
				out = append(out, item)
				continue
			}
			newID, keep := edit(id)
			if !keep || seen[newID] {
				changed = true
				continue
			}
			seen[newID] = true
			if newID != id {
				item = newID
				changed = true
			}
			out = append(out, item)
		}
		if changed {
			m[key] = out
		}
	}
	if kind == Extensions {
		apply(service, "extensions")
		return
	}
	for _, p := range c.pipelines() {
		if pipeline, ok := p.(map[string]any); ok {
			for _, key := range refKeys(kind) {
				apply(pipeline, key)
			}
		}
	}
}

func (c *OtelConfig) dropEmptySections() {
	for _, kind := range ComponentKinds {
		if section, ok := c.raw[string(kind)].(map[string]any); ok && len(section) == 0 {
			delete(c.raw, string(kind))
		}
	}
}

// refList reads a reference list. The collector also accepts a single ID or a comma-separated
// string in place of a list.
func refList(v any) []string {
	switch refs := v.(type) {
	case []any:
		out := make([]string, 0, len(refs))
		for _, ref := range refs {
			if id, ok := refID(ref); ok {
				out = append(out, id)
			}
		}
		return out
	case []string:
		return refs
	case string:
		var out []string
		for _, part := range strings.Split(refs, ",") {
			if part = strings.TrimSpace(part); part != "" {
				out = append(out, part)
			}
		}
		return out
	case xconfmap.ExpandedValue:
		return refList(refs.Value)
	}
	return nil
}

// refID returns the component ID that a list item holds. An item set from a ${...} reference whose
// value parses as a non-string, such as "true", is an xconfmap.ExpandedValue; its ID is the text of
// the value.
func refID(item any) (string, bool) {
	switch v := item.(type) {
	case string:
		return v, true
	case xconfmap.ExpandedValue:
		return v.Original, true
	}
	return "", false
}

// refProblems reports reference items that are not component IDs, and values that are neither a
// list nor a string. refList skips these, so Validate reports them.
func refProblems(v any) []string {
	switch refs := v.(type) {
	case nil, string, []string:
		return nil
	case []any:
		var out []string
		for i, item := range refs {
			if _, ok := refID(item); !ok {
				out = append(out, fmt.Sprintf("item %d is %T, not a component ID", i, item))
			}
		}
		return out
	case xconfmap.ExpandedValue:
		return refProblems(refs.Value)
	}
	return []string{fmt.Sprintf("is %T, not a list", v)}
}

func toAnyList(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

// deepMerge merges src into dst; values from src win on conflict.
func deepMerge(dst, src any) any {
	dstMap, ok1 := dst.(map[string]any)
	srcMap, ok2 := src.(map[string]any)
	if !ok1 || !ok2 {
		if src == nil {
			return dst
		}
		return src
	}
	out := maps.Clone(dstMap)
	for k, v := range srcMap {
		out[k] = deepMerge(out[k], v)
	}
	return out
}
