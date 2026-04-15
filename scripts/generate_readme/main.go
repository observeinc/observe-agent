package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

var displayNameOverrides = map[string]string{
	"dockerstatsreceiver":    "docker_stats",
	"k8sclusterreceiver":     "k8s_cluster",
	"healthcheckextension":   "health_check",
	"filestorage":            "file_storage",
	"memorylimiterprocessor": "memory_limiter",
}

var typeSuffixes = []string{
	"receiver", "processor", "exporter", "extension", "connector",
}

type component struct {
	GoMod string `yaml:"gomod"`
	Path  string `yaml:"path"`
}

type builderConfig struct {
	Dist struct {
		Version string `yaml:"version"`
	} `yaml:"dist"`
	Receivers  []component `yaml:"receivers"`
	Processors []component `yaml:"processors"`
	Exporters  []component `yaml:"exporters"`
	Extensions []component `yaml:"extensions"`
	Connectors []component `yaml:"connectors"`
}

type componentInfo struct {
	Anchor  string
	Display string
	URL     string
}

type templateData struct {
	GoVersion       string
	OtelVersion     string
	ComponentsTable string
}

func parseGoVersion(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	re := regexp.MustCompile(`^go\s+(\S+)`)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if m := re.FindStringSubmatch(scanner.Text()); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("go directive not found in %s", path)
}

func parseBuilderConfig(path string) (*builderConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg builderConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func deriveAnchor(gomod string) string {
	parts := strings.SplitN(gomod, " ", 2)
	pathParts := strings.Split(parts[0], "/")
	return pathParts[len(pathParts)-1]
}

func deriveDisplay(anchor string) string {
	if override, ok := displayNameOverrides[anchor]; ok {
		return override
	}
	for _, suffix := range typeSuffixes {
		if strings.HasSuffix(anchor, suffix) {
			return strings.TrimSuffix(anchor, suffix)
		}
	}
	return anchor
}

func deriveURL(gomod, localPath string) string {
	if localPath != "" {
		return localPath
	}
	parts := strings.SplitN(gomod, " ", 2)
	if len(parts) != 2 {
		return ""
	}
	modulePath := parts[0]
	version := parts[1]
	const otelCore = "go.opentelemetry.io/collector/"
	const otelContrib = "github.com/open-telemetry/opentelemetry-collector-contrib/"
	switch {
	case strings.HasPrefix(modulePath, otelCore):
		rest := strings.TrimPrefix(modulePath, otelCore)
		return fmt.Sprintf("https://github.com/open-telemetry/opentelemetry-collector/tree/%s/%s", version, rest)
	case strings.HasPrefix(modulePath, otelContrib):
		rest := strings.TrimPrefix(modulePath, otelContrib)
		return fmt.Sprintf("https://github.com/open-telemetry/opentelemetry-collector-contrib/tree/%s/%s", version, rest)
	default:
		return ""
	}
}

func buildComponentInfos(comps []component) []componentInfo {
	result := make([]componentInfo, 0, len(comps))
	for _, c := range comps {
		if c.GoMod == "" {
			continue
		}
		anchor := deriveAnchor(c.GoMod)
		display := deriveDisplay(anchor)
		url := deriveURL(c.GoMod, c.Path)
		result = append(result, componentInfo{Anchor: anchor, Display: display, URL: url})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Display < result[j].Display
	})
	return result
}

func buildComponentsTable(cfg *builderConfig) string {
	type column struct {
		header string
		items  []componentInfo
	}
	columns := []column{
		{"Receivers", buildComponentInfos(cfg.Receivers)},
		{"Processors", buildComponentInfos(cfg.Processors)},
		{"Exporters", buildComponentInfos(cfg.Exporters)},
		{"Extensions", buildComponentInfos(cfg.Extensions)},
		{"Connectors", buildComponentInfos(cfg.Connectors)},
	}

	widths := make([]int, len(columns))
	for i, col := range columns {
		widths[i] = len(col.header)
		for _, item := range col.items {
			cell := fmt.Sprintf("[%s][%s]", item.Display, item.Anchor)
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	maxRows := 0
	for _, col := range columns {
		if len(col.items) > maxRows {
			maxRows = len(col.items)
		}
	}

	var sb strings.Builder

	sb.WriteString("|")
	for i, col := range columns {
		sb.WriteString(fmt.Sprintf(" %-*s |", widths[i], col.header))
	}
	sb.WriteString("\n")

	sb.WriteString("|")
	for i := range columns {
		sb.WriteString(" " + strings.Repeat("-", widths[i]) + " |")
	}
	sb.WriteString("\n")

	for row := 0; row < maxRows; row++ {
		sb.WriteString("|")
		for i, col := range columns {
			cell := ""
			if row < len(col.items) {
				item := col.items[row]
				cell = fmt.Sprintf("[%s][%s]", item.Display, item.Anchor)
			}
			sb.WriteString(fmt.Sprintf(" %-*s |", widths[i], cell))
		}
		sb.WriteString("\n")
	}

	type link struct {
		anchor string
		url    string
	}
	var links []link
	seen := map[string]bool{}
	for _, col := range columns {
		for _, item := range col.items {
			if item.URL != "" && !seen[item.Anchor] {
				links = append(links, link{item.Anchor, item.URL})
				seen[item.Anchor] = true
			}
		}
	}
	sort.Slice(links, func(i, j int) bool {
		return links[i].anchor < links[j].anchor
	})

	sb.WriteString("\n")
	for _, l := range links {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", l.anchor, l.url))
	}

	return sb.String()
}

func main() {
	goVersion, err := parseGoVersion("go.mod")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing go.mod: %v\n", err)
		os.Exit(1)
	}

	cfg, err := parseBuilderConfig("builder-config.yaml")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing builder-config.yaml: %v\n", err)
		os.Exit(1)
	}

	data := templateData{
		GoVersion:       goVersion,
		OtelVersion:     cfg.Dist.Version,
		ComponentsTable: buildComponentsTable(cfg),
	}

	tmplContent, err := os.ReadFile("README.md.tmpl")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading README.md.tmpl: %v\n", err)
		os.Exit(1)
	}

	tmpl, err := template.New("readme").Parse(string(tmplContent))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing template: %v\n", err)
		os.Exit(1)
	}

	var buf bytes.Buffer
	buf.WriteString("<!-- Code generated by scripts/generate_readme.go. DO NOT EDIT. -->\n")
	if err := tmpl.Execute(&buf, data); err != nil {
		fmt.Fprintf(os.Stderr, "Error executing template: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile("README.md", buf.Bytes(), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing README.md: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("README.md generated successfully.")
}
