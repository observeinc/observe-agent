package connections

import (
	"context"
	"fmt"
	logger "observe-agent/cmd/commands/util"
	"os"
	"path/filepath"
	"runtime"
	"text/template"

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
}

func GetConfigFolderPath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		homedir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return filepath.Join(homedir, ".observe-agent/connections")
	case "windows":
		return os.ExpandEnv("$ProgramFiles\\Observe\\observe-agent\\connections")
	case "linux":
		return "/etc/observe-agent/connections"
	default:
		return "/etc/observe-agent/connections"
	}
}

func (c *ConnectionType) GetTemplateFilepath(tplFilename string) string {
	return filepath.Join(GetConfigFolderPath(), c.Name, tplFilename)
}

func (c *ConnectionType) RenderConfigTemplate(ctx context.Context, tmpDir string, tplFilename string, confValues any) (string, error) {
	tplPath := c.GetTemplateFilepath(tplFilename)
	tmpl, err := template.ParseFiles(tplPath)
	if err != nil {
		logger.FromCtx(ctx).Error("failed to parse config fragment template", zap.String("file", tplPath), zap.Error(err))
		return "", err
	}
	f, err := os.CreateTemp(tmpDir, fmt.Sprintf("*-%s", tplFilename))
	if err != nil {
		logger.FromCtx(ctx).Error("failed to create temporary config fragment file", zap.Error(err))
		return "", err
	}
	err = tmpl.Execute(f, confValues)
	if err != nil {
		logger.FromCtx(ctx).Error("failed to execute config fragment template", zap.Error(err))
		return "", err
	}
	return f.Name(), nil
}

func (c *ConnectionType) ProcessConfigFields(ctx context.Context, tmpDir string, rawConnConfig *viper.Viper, confValues any) ([]string, error) {
	paths := make([]string, 0)
	for _, field := range c.ConfigFields {
		val := rawConnConfig.GetBool(field.configYAMLPath)
		if val && field.colConfigFilePath != "" {
			configPath, _ := c.RenderConfigTemplate(ctx, tmpDir, field.colConfigFilePath, confValues)
			paths = append(paths, configPath)
		}
	}
	return paths, nil
}

func (c *ConnectionType) GetConfigFilePaths(ctx context.Context, tmpDir string) []string {
	var rawConnConfig = viper.Sub(c.Name)
	var configPaths []string
	if rawConnConfig == nil || !rawConnConfig.GetBool("enabled") {
		return configPaths
	}
	switch c.Type {
	case SelfMonitoringConnectionTypeName:
		conf := &SelfMonitoringConfig{}
		err := rawConnConfig.Unmarshal(conf)
		if err != nil {
			logger.FromCtx(ctx).Error("failed to unmarshal config", zap.String("connection", c.Name))
		}
		configPaths, _ = c.ProcessConfigFields(ctx, tmpDir, rawConnConfig, conf)
	case HostMonitoringConnectionTypeName:
		conf := &HostMonitoringConfig{}
		err := rawConnConfig.Unmarshal(conf)
		if err != nil {
			logger.FromCtx(ctx).Error("failed to unmarshal config", zap.String("connection", c.Name))
		}
		configPaths, _ = c.ProcessConfigFields(ctx, tmpDir, rawConnConfig, conf)
	}

	return configPaths
}

var AllConnectionTypes = []*ConnectionType{
	&HostMonitoringConnectionType,
	&SelfMonitoringConnectionType,
}
