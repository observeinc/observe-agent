package diagnose

import (
	"embed"
	"fmt"
	"os"

	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type ConfigTestResult struct {
	ConfigFile     string
	ParseSucceeded bool
	IsValid        bool
	Error          string
}

func checkConfig(v *viper.Viper) (bool, any, error) {
	configFile := v.ConfigFileUsed()
	if configFile == "" {
		return false, nil, fmt.Errorf("no config file defined")
	}
	contents, err := os.ReadFile(configFile)
	if err != nil {
		return false, nil, err
	}
	var conf config.AgentConfig
	if err = yaml.Unmarshal(contents, &conf); err != nil {
		return false, ConfigTestResult{
			ConfigFile:     configFile,
			ParseSucceeded: false,
			IsValid:        false,
			Error:          err.Error(),
		}, nil
	}
	if err = conf.Validate(); err != nil {
		return false, ConfigTestResult{
			ConfigFile:     configFile,
			ParseSucceeded: true,
			IsValid:        false,
			Error:          err.Error(),
		}, nil
	}
	return true, ConfigTestResult{
		ConfigFile:     configFile,
		ParseSucceeded: true,
		IsValid:        true,
	}, nil
}

const configcheckTemplate = "configcheck.tmpl"

var (
	//go:embed configcheck.tmpl
	configcheckTemplateFS embed.FS
)

func configDiagnostic() Diagnostic {
	return Diagnostic{
		check:        checkConfig,
		checkName:    "Config Check",
		templateName: configcheckTemplate,
		templateFS:   configcheckTemplateFS,
	}
}
