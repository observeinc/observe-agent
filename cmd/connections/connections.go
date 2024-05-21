package connections

import (
	"path/filepath"
	"runtime"

	"github.com/spf13/viper"
)

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
}

func GetConfigFolderPath() string {
	switch os := runtime.GOOS; os {
	case "darwin":
		return ""
	case "windows":
		return "%ProgramFiles%\\Observe\\observe-agent\\connections"
	case "linux":
		return "/etc/observe-agent/connections"
	default:
		return "/etc/observe-agent/connections"
	}
}

func (c ConnectionType) GetConfigFilePaths() []string {
	var rawConnConfig = viper.Sub(c.Name)
	configPaths := make([]string, 0)
	if rawConnConfig == nil || !rawConnConfig.GetBool("enabled") {
		return configPaths
	}
	for _, field := range c.ConfigFields {
		val := rawConnConfig.GetBool(field.configYAMLPath)
		if val && field.colConfigFilePath != "" {
			configPath := filepath.Join(GetConfigFolderPath(), c.Name, field.colConfigFilePath)
			configPaths = append(configPaths, configPath)
		}
	}
	return configPaths
}

var AllConnectionTypes = []*ConnectionType{
	&HostMonitoringConnectionType,
}
