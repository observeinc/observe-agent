package connections

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"path"
	"strings"
	"text/template"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig"
	"github.com/observeinc/observe-agent/internal/utils"
	"go.uber.org/zap"
)

// TempFilesFolder is the name prefix CleanupLegacyTempDirs uses to identify
// legacy observe-agent config directories in os.TempDir().
var TempFilesFolder = "observe-agent"

// RenderedConfigFragment is an otel config fragment held in memory and
// passed to the otelcol resolver via an inline `yaml:` URI.
type RenderedConfigFragment struct {
	// Name is a human-readable identifier used for logging and for the
	// `config print` command headers (e.g. `host_monitoring-host_metrics.yaml`).
	Name string
	// Content is the rendered YAML body of the fragment.
	Content string
}

type EnabledCheckFn func(*config.AgentConfig) bool

type BundledConfigFragment struct {
	enabledCheck      EnabledCheckFn
	colConfigFilePath string
}

type ConnectionType struct {
	Name                   string
	BundledConfigFragments []BundledConfigFragment
	EnabledCheck           EnabledCheckFn

	templateOverrides bundledconfig.ConfigTemplates
}

func (c *ConnectionType) getTemplate(tplName string) (*template.Template, error) {
	var fs embed.FS
	var ok bool
	if fs, ok = c.templateOverrides[tplName]; !ok {
		fs, ok = bundledconfig.SharedTemplateFS[tplName]
		if !ok {
			return nil, fmt.Errorf("template %s not found", tplName)
		}
	}
	return template.New(path.Base(tplName)).Funcs(utils.TemplateFuncs()).ParseFS(fs, tplName)
}

func renderBundledConfigTemplate(tmpl *template.Template, confValues any) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, confValues); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *ConnectionType) renderBundledConfigTemplate(ctx context.Context, tplName string, confValues any) (RenderedConfigFragment, error) {
	tmpl, err := c.getTemplate(c.Name + "/" + tplName)
	if err != nil {
		return RenderedConfigFragment{}, err
	}
	content, err := renderBundledConfigTemplate(tmpl, confValues)
	if err != nil {
		logger.FromCtx(ctx).Error("failed to execute config fragment template", zap.String("tplName", tplName), zap.Error(err))
		return RenderedConfigFragment{}, err
	}
	name := c.Name + "-" + strings.TrimSuffix(tplName, ".tmpl")
	return RenderedConfigFragment{Name: name, Content: content}, nil
}

func (c *ConnectionType) renderAllBundledConfigFragments(ctx context.Context, agentConfig *config.AgentConfig) ([]RenderedConfigFragment, error) {
	rendered := make([]RenderedConfigFragment, 0)
	for _, fragment := range c.BundledConfigFragments {
		if !fragment.enabledCheck(agentConfig) || fragment.colConfigFilePath == "" {
			continue
		}
		r, err := c.renderBundledConfigTemplate(ctx, fragment.colConfigFilePath, agentConfig)
		if err != nil {
			return nil, err
		}
		rendered = append(rendered, r)
	}
	return rendered, nil
}

func (c *ConnectionType) GetBundledConfigs(ctx context.Context, agentConfig *config.AgentConfig) ([]RenderedConfigFragment, error) {
	if !c.EnabledCheck(agentConfig) {
		return []RenderedConfigFragment{}, nil
	}

	return c.renderAllBundledConfigFragments(ctx, agentConfig)
}

type ConnectionTypeOption func(*ConnectionType)

func MakeConnectionType(name string, enabledCheck EnabledCheckFn, fragments []BundledConfigFragment, opts ...ConnectionTypeOption) *ConnectionType {
	var c = &ConnectionType{Name: name, EnabledCheck: enabledCheck, BundledConfigFragments: fragments}
	c.templateOverrides = bundledconfig.OverrideTemplates

	// Apply provided options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

func WithConfigTemplateOverrides(templateOverrides bundledconfig.ConfigTemplates) ConnectionTypeOption {
	return func(c *ConnectionType) {
		c.templateOverrides = templateOverrides
	}
}

func (c *ConnectionType) ApplyOptions(opts ...ConnectionTypeOption) *ConnectionType {
	for _, opt := range opts {
		opt(c)
	}
	return c
}
