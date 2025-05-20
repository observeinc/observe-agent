package diagnose

import (
	"embed"
	"errors"
	"fmt"
	"os"

	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
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
	if _, err := os.Stat(configFile); err != nil && errors.Is(err, os.ErrNotExist) {
		return false, nil, fmt.Errorf("config file %s does not exist", configFile)
	}
	agentConfig, err := config.AgentConfigFromViper(v)
	if err != nil {
		return false, ConfigTestResult{
			ConfigFile:     configFile,
			ParseSucceeded: false,
			IsValid:        false,
			Error:          err.Error(),
		}, nil
	}
	if err = agentConfig.Validate(); err != nil {
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
