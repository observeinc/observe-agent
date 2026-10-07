// Command generate_oss_examples turns the collector configuration that the Observe Agent renders
// (`observe-agent config --render-otel`) into a configuration that a plain OpenTelemetry Collector
// (otelcol-contrib) can run. It reads the rendered configuration on stdin and writes the example
// to stdout.
//
// Usage:
//
//	observe-agent config --render-otel --observe-config <reference> | \
//	  go run ./scripts/generate_oss_examples -reference <reference> > collector.yaml
package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

const header = `# OpenTelemetry Collector configuration for sending host and application telemetry to Observe.
#
# GENERATED FILE, DO NOT EDIT. This file is generated from the Observe Agent configuration in
# agent-reference-config.yaml. Run "make generate-oss-examples" to regenerate it.
#
# Set these environment variables before starting the collector:
#   OBSERVE_COLLECTION_URL  Your collection URL, e.g. https://123456789012.collect.observeinc.com
#   OBSERVE_TOKEN           An Observe ingest token
#   DEPLOYMENT_ENVIRONMENT  Fallback deployment.environment.name for telemetry that does not set one
`

// agentOnlyComponents lists components that the Observe Agent adds for its own operation. They
// are not part of the recommended setup for a plain OpenTelemetry Collector.
var agentOnlyComponents = map[string][]string{
	// Self-monitoring of the agent: log record counts and file stats of the agent's own files.
	"connectors": {"count"},
	"receivers":  {"filestats/agent", "nop"},
	"processors": {"filter/count", "resource/agent_instance"},
	"exporters":  {"debug", "nop"},
	// Persistent queue storage, which needs a directory the agent package creates.
	"extensions": {"file_storage"},
}

// referenceValues holds the placeholder values from the reference agent configuration that are
// replaced with environment variable references in the example.
type referenceValues struct {
	ObserveURL  string `yaml:"observe_url"`
	Token       string `yaml:"token"`
	ResourceEnv struct {
		Name string `yaml:"deployment.environment.name"`
	} `yaml:"resource_attributes"`
}

func main() {
	referencePath := flag.String("reference", "", "path to the reference observe-agent configuration")
	flag.Parse()
	if err := run(*referencePath, os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(referencePath string, in io.Reader, out io.Writer) error {
	refBytes, err := os.ReadFile(referencePath)
	if err != nil {
		return fmt.Errorf("reading reference config: %w", err)
	}
	var ref referenceValues
	if err := yaml.Unmarshal(refBytes, &ref); err != nil {
		return fmt.Errorf("parsing reference config: %w", err)
	}
	rendered, err := io.ReadAll(in)
	if err != nil {
		return fmt.Errorf("reading rendered config: %w", err)
	}
	example, err := generate(rendered, ref)
	if err != nil {
		return err
	}
	_, err = out.Write(example)
	return err
}

func generate(rendered []byte, ref referenceValues) ([]byte, error) {
	var cfg map[string]any
	if err := yaml.Unmarshal(rendered, &cfg); err != nil {
		return nil, fmt.Errorf("parsing rendered config: %w", err)
	}
	if ref.ObserveURL == "" || ref.Token == "" || ref.ResourceEnv.Name == "" {
		return nil, fmt.Errorf("reference config must set observe_url, token, and resource_attributes.deployment.environment.name")
	}

	removeAgentOnlyComponents(cfg)

	replacer := strings.NewReplacer(
		strings.TrimSuffix(ref.ObserveURL, "/"), "${env:OBSERVE_COLLECTION_URL}",
		ref.Token, "${env:OBSERVE_TOKEN}",
	)
	cfg = replaceStrings(cfg, func(s string) string {
		if s == ref.ResourceEnv.Name {
			return "${env:DEPLOYMENT_ENVIRONMENT}"
		}
		return replacer.Replace(s)
	}).(map[string]any)

	var buf bytes.Buffer
	buf.WriteString(header)
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(cfg); err != nil {
		return nil, fmt.Errorf("encoding example: %w", err)
	}
	if bytes.Contains(buf.Bytes(), []byte(ref.Token)) || bytes.Contains(buf.Bytes(), []byte(strings.TrimSuffix(ref.ObserveURL, "/"))) {
		return nil, fmt.Errorf("generated example still contains the reference token or collection URL")
	}
	return buf.Bytes(), nil
}

// removeAgentOnlyComponents deletes agentOnlyComponents from the configuration and from every
// pipeline, then drops pipelines that are left without receivers or exporters.
func removeAgentOnlyComponents(cfg map[string]any) {
	removed := map[string]bool{}
	for section, ids := range agentOnlyComponents {
		components, _ := cfg[section].(map[string]any)
		for _, id := range ids {
			delete(components, id)
			removed[id] = true
		}
		if components != nil && len(components) == 0 {
			delete(cfg, section)
		}
	}

	service, _ := cfg["service"].(map[string]any)
	service["extensions"] = withoutRemoved(service["extensions"], removed)
	if len(service["extensions"].([]any)) == 0 {
		delete(service, "extensions")
	}
	pipelines, _ := service["pipelines"].(map[string]any)
	for name, p := range pipelines {
		pipeline := p.(map[string]any)
		for _, key := range []string{"receivers", "processors", "exporters"} {
			pipeline[key] = withoutRemoved(pipeline[key], removed)
		}
		if len(pipeline["receivers"].([]any)) == 0 || len(pipeline["exporters"].([]any)) == 0 {
			delete(pipelines, name)
		}
	}
}

func withoutRemoved(list any, removed map[string]bool) []any {
	items, _ := list.([]any)
	kept := []any{}
	for _, item := range items {
		if id, ok := item.(string); !ok || !removed[id] {
			kept = append(kept, item)
		}
	}
	return kept
}

func replaceStrings(v any, replace func(string) string) any {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			t[k] = replaceStrings(val, replace)
		}
		return t
	case []any:
		for i, val := range t {
			t[i] = replaceStrings(val, replace)
		}
		return t
	case string:
		return replace(t)
	default:
		return v
	}
}
