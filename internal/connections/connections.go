package connections

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	logger "github.com/observeinc/observe-agent/internal/commands/util"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var TempFilesFolder = "observe-agent"

type ConfigFieldHandler interface {
	GenerateCollectorConfigFragment() interface{}
}

type CollectorConfigFragment struct {
	configYAMLPath    string
	colConfigFilePath string
}

type ConnectionType struct {
	Name         string
	ConfigFields []CollectorConfigFragment
	Type         string

	configFolderPath string
	getConfig        func() *viper.Viper
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

func (c *ConnectionType) ProcessConfigFields(ctx context.Context, tmpDir string, rawConnConfig *viper.Viper, confValues any) ([]string, error) {
	paths := make([]string, 0)
	for _, field := range c.ConfigFields {
		val := rawConnConfig.GetBool(field.configYAMLPath)
		if val && field.colConfigFilePath != "" {
			configPath, err := c.RenderConfigTemplate(ctx, tmpDir, field.colConfigFilePath, confValues)
			if err != nil {
				return nil, err
			}
			paths = append(paths, configPath)
		}
	}
	return paths, nil
}

func (c *ConnectionType) GetConfigFilePaths(ctx context.Context, tmpDir string) ([]string, error) {
	var rawConnConfig = c.getConfig()
	var configPaths []string
	if rawConnConfig == nil || !rawConnConfig.GetBool("enabled") {
		return configPaths, nil
	}
	switch c.Type {
	case SelfMonitoringConnectionTypeName:
		conf := &config.SelfMonitoringConfig{}
		err := rawConnConfig.Unmarshal(conf)
		if err != nil {
			logger.FromCtx(ctx).Error("failed to unmarshal config", zap.String("connection", c.Name))
			return nil, err
		}
		configPaths, err = c.ProcessConfigFields(ctx, tmpDir, rawConnConfig, conf)
		if err != nil {
			return nil, err
		}
	case HostMonitoringConnectionTypeName:
		conf := &config.HostMonitoringConfig{}
		err := rawConnConfig.Unmarshal(conf)
		if err != nil {
			logger.FromCtx(ctx).Error("failed to unmarshal config", zap.String("connection", c.Name))
			return nil, err
		}
		configPaths, err = c.ProcessConfigFields(ctx, tmpDir, rawConnConfig, conf)
		if err != nil {
			return nil, err
		}
	default:
		logger.FromCtx(ctx).Error("unknown connection type", zap.String("type", c.Type))
		return nil, fmt.Errorf("unknown connection type %s", c.Type)
	}
	return configPaths, nil
}

type ConnectionTypeOption func(*ConnectionType)

func MakeConnectionType(Name string, ConfigFields []CollectorConfigFragment, Type string, opts ...ConnectionTypeOption) *ConnectionType {
	var c = &ConnectionType{Name: Name, ConfigFields: ConfigFields, Type: Type}
	c.getConfig = func() *viper.Viper {
		return viper.Sub(c.Name)
	}
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

func WithGetConfig(getConfig func() *viper.Viper) ConnectionTypeOption {
	return func(c *ConnectionType) {
		c.getConfig = getConfig
	}
}

var AllConnectionTypes = []*ConnectionType{
	HostMonitoringConnectionType,
	SelfMonitoringConnectionType,
}
