package connections

import "github.com/spf13/viper"

type ConfigFieldHandler interface {
	GenerateCollectorConfigFragment() interface{}
}

type CollectorConfigFragment struct {
	configYAMLPath    string
	colConfigFilePath string
}

type ConnectionType[T interface{}] struct {
	Name         string
	Config       T
	ConfigFields []CollectorConfigFragment
}

func (c ConnectionType[T]) GetConfigFilePaths() []string {
	var rawConnConfig = viper.Sub(c.Name)
	configPaths := make([]string, 0)
	if rawConnConfig == nil || rawConnConfig.GetBool("enabled") != true {
		return configPaths
	}
	for _, field := range c.ConfigFields {
		val := rawConnConfig.GetBool(field.configYAMLPath)
		if val && field.colConfigFilePath != "" {
			configPaths = append(configPaths, field.colConfigFilePath)
		}
	}
	return configPaths
}
