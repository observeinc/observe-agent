package connections

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"go.uber.org/zap"
)

var TempFilesFolder = "observe-agent"

type EnabledCheckFn func(*config.AgentConfig) bool

type ConfigFieldHandler interface {
	GenerateCollectorConfigFragment() interface{}
}

type CollectorConfigFragment struct {
	enabledCheck      EnabledCheckFn
	colConfigFilePath string
}

type ConnectionType struct {
	Name         string
	ConfigFields []CollectorConfigFragment
	EnabledCheck EnabledCheckFn

	configFolderPath string
}

func (c *ConnectionType) GetTemplateFilepath(tplFilename string) string {
	return filepath.Join(c.configFolderPath, c.Name, tplFilename)
}

func RenderConfigTemplate(ctx context.Context, tmpDir string, tplPath string, confValues any) (string, error) {
	_, tplFilename := filepath.Split(tplPath)
	tmpl, err := template.New("").Funcs(GetTemplateFuncMap()).ParseFiles(tplPath)
	if err != nil {
		logger.FromCtx(ctx).Error("failed to parse config fragment template", zap.String("file", tplPath), zap.Error(err))
		return "", err
	}
	f, err := os.CreateTemp(tmpDir, fmt.Sprintf("*-%s", tplFilename))
	if err != nil {
		logger.FromCtx(ctx).Error("failed to create temporary config fragment file", zap.String("file", tplPath), zap.Error(err))
		return "", err
	}
	err = tmpl.ExecuteTemplate(f, tplFilename, confValues)
	if err != nil {
		logger.FromCtx(ctx).Error("failed to execute config fragment template", zap.String("file", tplPath), zap.Error(err))
		return "", err
	}
	return f.Name(), nil
}

func (c *ConnectionType) RenderConfigTemplate(ctx context.Context, tmpDir string, tplFilename string, confValues any) (string, error) {
	tplPath := c.GetTemplateFilepath(tplFilename)
	return RenderConfigTemplate(ctx, tmpDir, tplPath, confValues)
}

func (c *ConnectionType) ProcessConfigFields(ctx context.Context, tmpDir string, agentConfig *config.AgentConfig) ([]string, error) {
	paths := make([]string, 0)
	for _, field := range c.ConfigFields {
		if !field.enabledCheck(agentConfig) || field.colConfigFilePath == "" {
			continue
		}
		configPath, err := c.RenderConfigTemplate(ctx, tmpDir, field.colConfigFilePath, agentConfig)
		if err != nil {
			return nil, err
		}
		paths = append(paths, configPath)
	}
	return paths, nil
}

func (c *ConnectionType) GetConfigFilePaths(ctx context.Context, tmpDir string, agentConfig *config.AgentConfig) ([]string, error) {
	if !c.EnabledCheck(agentConfig) {
		return []string{}, nil
	}

	configPaths, err := c.ProcessConfigFields(ctx, tmpDir, agentConfig)
	if err != nil {
		return nil, err
	}
	return configPaths, nil
}

type ConnectionTypeOption func(*ConnectionType)

func MakeConnectionType(name string, enabledCheck EnabledCheckFn, configFields []CollectorConfigFragment, opts ...ConnectionTypeOption) *ConnectionType {
	var c = &ConnectionType{Name: name, EnabledCheck: enabledCheck, ConfigFields: configFields}
	c.configFolderPath = GetConfigFragmentFolderPath()

	// Apply provided options
	for _, opt := range opts {
		opt(c)
	}

	return c
}

func WithConfigFolderPath(configFolderPath string) ConnectionTypeOption {
	return func(c *ConnectionType) {
		c.configFolderPath = configFolderPath
	}
}

func (c *ConnectionType) ApplyOptions(opts ...ConnectionTypeOption) *ConnectionType {
	for _, opt := range opts {
		opt(c)
	}
	return c
}
