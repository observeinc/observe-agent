package connections

import "github.com/spf13/viper"

type ConfigFieldHandler interface {
	GenerateCollectorConfigFragment() interface{}
}

type CollectorConfigFragment struct {
	configYAMLPath    string
	colConfigFilePath string
	required          bool
}

type ConnectionType[T interface{}] struct {
	Name         string
	Config       T
	ConfigFields []CollectorConfigFragment
}

// var hostMonConfig HostMonitoringConfig
// if err := rawHostMonConfig.Unmarshal(&hostMonConfig); err != nil {
// 	fmt.Println(err)
// 	return
// }
// fmt.Println(hostMonConfig)

func (c ConnectionType[T]) GetConfigFilePaths() []string {
	var rawConnConfig = viper.Sub(c.Name)
	configPaths := make([]string, 0)
	if rawConnConfig == nil {
		return configPaths
	}
	for _, field := range c.ConfigFields {
		val := rawConnConfig.GetBool(field.configYAMLPath)
		if val == true {
			configPaths = append(configPaths, field.colConfigFilePath)
		} else {
			if field.required {
				return configPaths
			}
		}
	}
	return configPaths
}
