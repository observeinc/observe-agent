package connections

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path"
	"strings"
	"text/template"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig"
	"go.uber.org/zap"
)

var TempFilesFolder = "observe-agent"

type EnabledCheckFn func(*config.AgentConfig) bool

type BundledConfigFragment struct {
	enabledCheck      EnabledCheckFn
	colConfigFilePath string
}

type ConfigOverrides = map[string]embed.FS

type ConnectionType struct {
	Name                   string
	BundledConfigFragments []BundledConfigFragment
	EnabledCheck           EnabledCheckFn

	templateOverrides ConfigOverrides
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
	return template.New(path.Base(tplName)).Funcs(TemplateFuncMap).ParseFS(fs, tplName)
}

func renderBundledConfigTemplate(ctx context.Context, tmpDir string, outFileName string, tmpl *template.Template, confValues any) (string, error) {
	f, err := os.CreateTemp(tmpDir, fmt.Sprintf("*-%s", outFileName))
	if err != nil {
		logger.FromCtx(ctx).Error("failed to create temporary config fragment file", zap.String("fileName", outFileName), zap.Error(err))
		return "", err
	}
	err = tmpl.Execute(f, confValues)
	if err != nil {
		logger.FromCtx(ctx).Error("failed to execute config fragment template", zap.String("fileName", outFileName), zap.Error(err))
		return "", err
	}
	return f.Name(), nil
}

func (c *ConnectionType) renderBundledConfigTemplate(ctx context.Context, tmpDir string, tplName string, confValues any) (string, error) {
	tmpl, err := c.getTemplate(c.Name + "/" + tplName)
	if err != nil {
		fmt.Printf("TODO err1: %s\n", err.Error())
		return "", err
	}
	outFileName := c.Name + "-" + strings.TrimSuffix(tplName, ".tmpl")
	return renderBundledConfigTemplate(ctx, tmpDir, outFileName, tmpl, confValues)
}

func (c *ConnectionType) renderAllBundledConfigFragments(ctx context.Context, tmpDir string, agentConfig *config.AgentConfig) ([]string, error) {
	paths := make([]string, 0)
	for _, fragment := range c.BundledConfigFragments {
		if !fragment.enabledCheck(agentConfig) || fragment.colConfigFilePath == "" {
			continue
		}
		configPath, err := c.renderBundledConfigTemplate(ctx, tmpDir, fragment.colConfigFilePath, agentConfig)
		if err != nil {
			return nil, err
		}
		paths = append(paths, configPath)
	}
	return paths, nil
}

func (c *ConnectionType) GetBundledConfigs(ctx context.Context, tmpDir string, agentConfig *config.AgentConfig) ([]string, error) {
	if !c.EnabledCheck(agentConfig) {
		return []string{}, nil
	}

	configPaths, err := c.renderAllBundledConfigFragments(ctx, tmpDir, agentConfig)
	if err != nil {
		return nil, err
	}
	return configPaths, nil
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

func WithConfigTemplateOverrides(templateOverrides ConfigOverrides) ConnectionTypeOption {
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
