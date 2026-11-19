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
//	go run ./scripts/generate_oss_examples --agent-config <file> [--otel-config <file>] \
//	  --config-mode linux --pipelines traces/forward,logs/forward [--extensions <ids>] --out <file>
package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"slices"
	"strconv"
	"strings"

	"github.com/observeinc/observe-agent/internal/commands/config"
	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	agentconfig "github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/pflag"
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
	flags := pflag.NewFlagSet("generate_oss_examples", pflag.ExitOnError)
	flags.String("agent-config", "", "observe-agent configuration to render (required)")
	flags.String("otel-config", "", "OpenTelemetry Collector configuration to merge on top of the bundled configuration")
	flags.String("config-mode", "linux", "bundled configuration mode: linux, docker, macos, or windows")
	flags.StringSlice("pipelines", nil, "comma-separated pipelines to keep (required)")
	flags.StringSlice("extensions", nil, "comma-separated extensions to keep")
	flags.String("out", "", "file to write the example to (required)")
	_ = flags.Parse(os.Args[1:])

	// A separate instance from the global one that root.InitConfig loads the agent config into.
	v := viper.New()
	if err := v.BindPFlags(flags); err != nil {
		fmt.Fprintln(os.Stderr, "generate_oss_examples:", err)
		os.Exit(1)
	}
	opts := options{
		agentConfig: v.GetString("agent-config"),
		otelConfig:  v.GetString("otel-config"),
		configMode:  v.GetString("config-mode"),
		pipelines:   v.GetStringSlice("pipelines"),
		extensions:  v.GetStringSlice("extensions"),
		out:         v.GetString("out"),
	}

	if err := run(opts); err != nil {
		fmt.Fprintln(os.Stderr, "generate_oss_examples:", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	if opts.agentConfig == "" || opts.out == "" || len(opts.pipelines) == 0 {
		return errors.New("--agent-config, --pipelines, and --out are required")
	}
	// root.InitConfig exits the process on an invalid mode and only prints config read errors, so
	// check both before rendering.
	if !slices.Contains([]string{"linux", "docker", "macos", "windows"}, strings.ToLower(opts.configMode)) {
		return fmt.Errorf("invalid --config-mode %q: use linux, docker, macos, or windows", opts.configMode)
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
	otelCfg, err := agentconfig.NewOtelConfigFromYAML(rendered)
	if err != nil {
		return fmt.Errorf("parsing rendered config: %w", err)
	}
	dropAgentInstanceID(otelCfg)
	if err := prune(otelCfg, opts.pipelines, opts.extensions); err != nil {
		return err
	}

	renderedExample, err := otelCfg.ToYAML()
	if err != nil {
		return fmt.Errorf("encoding example: %w", err)
	}
	body := bytes.NewBufferString(renderedExample)
	if err := checkNoCredentials(body.Bytes(), opts.agentConfig); err != nil {
		return err
	}
	if err := checkNoAgentFields(body.Bytes()); err != nil {
		return err
	}

	version, err := collectorVersion()
	if err != nil {
		return err
	}

	var out bytes.Buffer
	out.WriteString(header)
	fmt.Fprintf(&out, "#\n# Run it with otelcol-contrib %s, the collector version the Observe Agent is built with.\n", version)
	if vars := envVars(body.Bytes()); len(vars) > 0 {
		out.WriteString("#\n# Set these environment variables before starting the collector; it doesn't start without them\n# (see examples/oss-collector/README.md): " + strings.Join(vars, ", ") + "\n")
	}
	out.Write(body.Bytes())
	return writeAtomically(opts.out, out.Bytes())
}

// collectorVersion returns the OpenTelemetry Collector version that the generator, like the agent,
// is built with.
func collectorVersion() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.New("reading the collector version: no build information")
	}
	for _, dep := range info.Deps {
		if dep.Path == "go.opentelemetry.io/collector/otelcol" {
			if dep.Replace != nil {
				dep = dep.Replace
			}
			return strings.TrimPrefix(dep.Version, "v"), nil
		}
	}
	return "", errors.New("reading the collector version: go.opentelemetry.io/collector/otelcol is not a dependency")
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

// prune keeps only the given pipelines and extensions, and deletes every component that the kept
// pipelines don't reference. A connector is only kept if one kept pipeline exports to it and
// another receives from it; otherwise it is also removed from the pipelines that list it.
func prune(cfg *agentconfig.OtelConfig, keepPipelines, keepExtensions []string) error {
	if err := cfg.RetainPipelines(keepPipelines...); err != nil {
		return fmt.Errorf("rendered config: %w", err)
	}
	if err := cfg.RetainExtensions(keepExtensions...); err != nil {
		return fmt.Errorf("rendered config: %w", err)
	}
	cfg.RemoveUnusedComponents()
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("rendered config: %w", err)
	}
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
			return fmt.Errorf("example contains the agent config's %s; set the values that reference it in --otel-config", name)
		}
	}
	return nil
}

// agentInstanceIDKey identifies an Observe Agent by an ID the agent generates for itself, so a
// plain collector has nothing to set it to.
const agentInstanceIDKey = "observe.agent.instance.id"

// dropAgentInstanceID removes the agent's instance ID from the internal telemetry resource and
// from the labels the internal metrics carry.
func dropAgentInstanceID(cfg *agentconfig.OtelConfig) {
	telemetry := cfg.Telemetry()
	telemetry.Delete("resource", agentInstanceIDKey)
	if resource, ok := telemetry.Get("resource"); ok {
		if m, _ := resource.(map[string]any); len(m) == 0 {
			telemetry.Delete("resource")
		}
	}
	readers, _ := telemetry.Get("metrics", "readers")
	readerList, _ := readers.([]any)
	for i := range readerList {
		labels := []string{"metrics", "readers", strconv.Itoa(i), "pull", "exporter", "prometheus", "with_resource_constant_labels"}
		v, ok := telemetry.Get(append(labels, "included")...)
		if !ok {
			continue
		}
		included, _ := v.([]any)
		included = slices.DeleteFunc(slices.Clone(included), func(v any) bool { return v == agentInstanceIDKey })
		if len(included) == 0 {
			telemetry.Delete(labels...)
		} else {
			_ = telemetry.Set(included, append(labels, "included")...)
		}
	}
}

var agentFieldPattern = regexp.MustCompile(`observe\.agent\.[A-Za-z0-9_.]+|OBSERVE_AGENT_[A-Z0-9_]+`)

// checkNoAgentFields fails if the example still contains a field or variable that only the
// Observe Agent sets.
func checkNoAgentFields(example []byte) error {
	if field := agentFieldPattern.Find(example); field != nil {
		return fmt.Errorf("example contains %q, which only the Observe Agent sets", field)
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
