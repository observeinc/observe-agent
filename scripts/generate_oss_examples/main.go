// Command generate_oss_examples generates a configuration for a plain OpenTelemetry Collector
// (otelcol-contrib) from the Observe Agent's bundled configuration.
//
// It renders the collector configuration the agent would run with the given agent config and an
// optional OpenTelemetry Collector config merged on top (as with the agent's --config flag), keeps
// only the requested pipelines and extensions, and deletes every component they don't reference.
//
// The optional collector config is where example-specific values go. Write "$${env:VAR}" for a
// value that the example should read from the environment: rendering turns "$$" into "$", so the
// example contains "${env:VAR}".
//
// Usage:
//
//	go run ./scripts/generate_oss_examples -agent-config <file> [-otel-config <file>] \
//	  -config-mode linux -pipelines traces/forward,logs/forward [-extensions <ids>] -out <file>
package main

import (
	"bytes"
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/observeinc/observe-agent/internal/commands/config"
	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const header = `# OpenTelemetry Collector configuration for sending telemetry to Observe.
#
# GENERATED FILE, DO NOT EDIT. Run "make generate-oss-examples" to regenerate it from the Observe
# Agent configuration.
`

type options struct {
	agentConfig string
	otelConfig  string
	configMode  string
	pipelines   []string
	extensions  []string
	out         string
}

func main() {
	var opts options
	var pipelines, extensions string
	flag.StringVar(&opts.agentConfig, "agent-config", "", "observe-agent configuration to render (required)")
	flag.StringVar(&opts.otelConfig, "otel-config", "", "OpenTelemetry Collector configuration to merge on top of the bundled configuration")
	flag.StringVar(&opts.configMode, "config-mode", "linux", "bundled configuration mode: linux, docker, macos, or windows")
	flag.StringVar(&pipelines, "pipelines", "", "comma-separated pipelines to keep (required)")
	flag.StringVar(&extensions, "extensions", "", "comma-separated extensions to keep")
	flag.StringVar(&opts.out, "out", "", "file to write the example to (required)")
	flag.Parse()
	opts.pipelines = splitList(pipelines)
	opts.extensions = splitList(extensions)

	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "generate_oss_examples:", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	if opts.agentConfig == "" || opts.out == "" || len(opts.pipelines) == 0 {
		return errors.New("-agent-config, -pipelines, and -out are required")
	}
	// root.InitConfig exits the process on an invalid mode and only prints config read errors, so
	// check both before rendering.
	if !slices.Contains([]string{"linux", "docker", "macos", "windows"}, strings.ToLower(opts.configMode)) {
		return fmt.Errorf("invalid -config-mode %q: use linux, docker, macos, or windows", opts.configMode)
	}
	for _, path := range []string{opts.agentConfig, opts.otelConfig} {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			return fmt.Errorf("reading config: %w", err)
		}
	}
	rendered, err := render(opts)
	if err != nil {
		return err
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(rendered, &cfg); err != nil {
		return fmt.Errorf("parsing rendered config: %w", err)
	}
	if err := prune(cfg, opts.pipelines, opts.extensions); err != nil {
		return err
	}

	var body bytes.Buffer
	enc := yaml.NewEncoder(&body)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return fmt.Errorf("encoding example: %w", err)
	}
	if err := checkNoCredentials(body.Bytes(), opts.agentConfig); err != nil {
		return err
	}

	var out bytes.Buffer
	out.WriteString(header)
	if vars := envVars(body.Bytes()); len(vars) > 0 {
		out.WriteString("#\n# Set these environment variables before starting the collector; it doesn't start without them\n# (see examples/oss-collector/README.md): " + strings.Join(vars, ", ") + "\n")
	}
	out.Write(body.Bytes())
	return writeAtomically(opts.out, out.Bytes())
}

var envVarPattern = regexp.MustCompile(`\$\{env:([A-Za-z0-9_]+)\}`)

// envVars returns the sorted names of the environment variables the configuration references.
func envVars(cfg []byte) []string {
	var names []string
	for _, m := range envVarPattern.FindAllSubmatch(cfg, -1) {
		if name := string(m[1]); !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names
}

// render returns the collector configuration the agent would run, with environment variables
// resolved.
func render(opts options) ([]byte, error) {
	localDir, err := os.MkdirTemp("", "generate-oss-examples")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(localDir)
	// The agent stores its instance ID here; keep it out of the default system path.
	viper.Set("agent_local_file_path", filepath.Join(localDir, "agent_data.json"))

	flags := root.RootCmd.PersistentFlags()
	settings := [][2]string{{"observe-config", opts.agentConfig}, {"config-mode", opts.configMode}}
	if opts.otelConfig != "" {
		absPath, err := filepath.Abs(opts.otelConfig)
		if err != nil {
			return nil, err
		}
		settings = append(settings, [2]string{"config", "file:" + absPath})
	}
	for _, s := range settings {
		if err := flags.Set(s[0], s[1]); err != nil {
			return nil, fmt.Errorf("setting --%s: %w", s[0], err)
		}
	}
	root.InitConfig()

	var buf bytes.Buffer
	ctx := logger.WithCtx(context.Background(), logger.GetNop())
	if err := config.PrintShortOtelConfig(ctx, &buf); err != nil {
		return nil, fmt.Errorf("rendering collector config: %w", err)
	}
	return buf.Bytes(), nil
}

var componentSections = []string{"receivers", "processors", "exporters", "connectors"}

// prune keeps only the given pipelines and extensions, and deletes every component that the kept
// pipelines don't reference. A connector is only kept if one kept pipeline exports to it and
// another receives from it; otherwise it is also removed from the pipelines that list it.
func prune(cfg map[string]any, keepPipelines, keepExtensions []string) error {
	service, ok := cfg["service"].(map[string]any)
	if !ok {
		return errors.New("rendered config has no service section")
	}
	allPipelines, ok := service["pipelines"].(map[string]any)
	if !ok {
		return errors.New("rendered config has no service.pipelines section")
	}

	pipelines := map[string]map[string][]string{}
	for _, name := range keepPipelines {
		raw, ok := allPipelines[name].(map[string]any)
		if !ok {
			return fmt.Errorf("pipeline %q is not in the rendered config", name)
		}
		pipeline := map[string][]string{}
		for _, key := range []string{"receivers", "processors", "exporters"} {
			ids, err := stringList(raw[key])
			if err != nil {
				return fmt.Errorf("pipeline %q %s: %w", name, key, err)
			}
			pipeline[key] = ids
		}
		pipelines[name] = pipeline
	}

	connectors, _ := cfg["connectors"].(map[string]any)
	isConnector := func(id string) bool { _, ok := connectors[id]; return ok }
	for {
		exported, received := map[string]bool{}, map[string]bool{}
		for _, p := range pipelines {
			for _, id := range p["exporters"] {
				exported[id] = true
			}
			for _, id := range p["receivers"] {
				received[id] = true
			}
		}
		unusable := func(id string) bool { return isConnector(id) && !(exported[id] && received[id]) }
		changed := false
		for name, p := range pipelines {
			for _, key := range []string{"receivers", "exporters"} {
				kept := slices.DeleteFunc(slices.Clone(p[key]), unusable)
				if len(kept) != len(p[key]) {
					p[key] = kept
					changed = true
				}
				if len(kept) == 0 {
					return fmt.Errorf("pipeline %q has no %s left after removing connectors with no kept pipeline on the other side", name, key)
				}
			}
		}
		if !changed {
			break
		}
	}

	used := map[string]map[string]bool{}
	for _, section := range componentSections {
		used[section] = map[string]bool{}
	}
	for _, p := range pipelines {
		for key, ids := range p {
			for _, id := range ids {
				if key != "processors" && isConnector(id) {
					used["connectors"][id] = true
				} else {
					used[key][id] = true
				}
			}
		}
	}
	for _, section := range componentSections {
		components, _ := cfg[section].(map[string]any)
		for id := range components {
			if !used[section][id] {
				delete(components, id)
			}
		}
		for id := range used[section] {
			if _, ok := components[id]; !ok {
				return fmt.Errorf("pipelines reference %s %q, which the rendered config doesn't define", strings.TrimSuffix(section, "s"), id)
			}
		}
		if len(components) == 0 {
			delete(cfg, section)
		}
	}

	extensions, _ := cfg["extensions"].(map[string]any)
	for id := range extensions {
		if !slices.Contains(keepExtensions, id) {
			delete(extensions, id)
		}
	}
	for _, id := range keepExtensions {
		if _, ok := extensions[id]; !ok {
			return fmt.Errorf("extension %q is not in the rendered config", id)
		}
	}
	if len(extensions) == 0 {
		delete(cfg, "extensions")
		delete(service, "extensions")
	} else {
		service["extensions"] = toAnyList(keepExtensions)
	}

	keptPipelines := map[string]any{}
	for name, p := range pipelines {
		keptPipelines[name] = map[string]any{
			"receivers":  toAnyList(p["receivers"]),
			"processors": toAnyList(p["processors"]),
			"exporters":  toAnyList(p["exporters"]),
		}
	}
	service["pipelines"] = keptPipelines
	return nil
}

// checkNoCredentials fails if the example still contains the collection URL or token from the
// agent config, for example because an exporter that the otel config doesn't override kept the
// rendered value.
func checkNoCredentials(example []byte, agentConfigPath string) error {
	raw, err := os.ReadFile(agentConfigPath)
	if err != nil {
		return err
	}
	var agentCfg struct {
		ObserveURL string `yaml:"observe_url"`
		Token      string `yaml:"token"`
	}
	if err := yaml.Unmarshal(raw, &agentCfg); err != nil {
		return fmt.Errorf("parsing agent config: %w", err)
	}
	for name, value := range map[string]string{"observe_url": strings.TrimSuffix(agentCfg.ObserveURL, "/"), "token": agentCfg.Token} {
		if value != "" && bytes.Contains(example, []byte(value)) {
			return fmt.Errorf("example contains the agent config's %s; set the values that reference it in -otel-config", name)
		}
	}
	return nil
}

// newExampleMode is the mode of a newly generated example: it is committed documentation, so
// everyone may read it.
const newExampleMode os.FileMode = 0o644

// writeAtomically replaces path only once the full content is written, so a failed run leaves
// the existing file untouched. A replaced file keeps its mode.
func writeAtomically(path string, content []byte) error {
	mode := newExampleMode
	if info, err := os.Stat(path); err == nil {
		mode = info.Mode().Perm()
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(content); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

func stringList(v any) ([]string, error) {
	if v == nil {
		return nil, nil
	}
	items, ok := v.([]any)
	if !ok {
		return nil, fmt.Errorf("expected a list, got %T", v)
	}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		id, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("expected component IDs, got %T", item)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func toAnyList(ids []string) []any {
	out := make([]any, len(ids))
	for i, id := range ids {
		out[i] = id
	}
	return out
}

func splitList(s string) []string {
	var out []string
	for _, item := range strings.Split(s, ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
