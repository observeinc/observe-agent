package connections

import (
	"os"
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
