package diagnose

import (
	"embed"
	"errors"
	"fmt"
	"os"

	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type ConfigTestResult struct {
	ConfigFile     string
	ParseSucceeded bool
	IsValid        bool
	Error          string
}

func checkConfig(v *viper.Viper) (bool, any, error) {
	// Ensure there is an observe-agent config file.
	configFile := v.ConfigFileUsed()
	if configFile == "" {
		return false, nil, fmt.Errorf("no config file defined")
	}
	if _, err := os.Stat(configFile); err != nil && errors.Is(err, os.ErrNotExist) {
		return false, nil, fmt.Errorf("config file %s does not exist", configFile)
	}

	// Ensure the file is valid yaml.
	contents, err := os.ReadFile(configFile)
	if err != nil {
		return false, nil, fmt.Errorf("error reading config file %s: %w", configFile, err)
	}
	var yamlMap map[string]any
	if err = yaml.Unmarshal(contents, &yamlMap); err != nil {
		return false, ConfigTestResult{
			ConfigFile:     configFile,
			ParseSucceeded: false,
			IsValid:        false,
			Error:          err.Error(),
		}, nil
	}

	// Ensure the agent config can be loaded via viper.
	agentConfig, err := config.AgentConfigFromViper(v)
	if err != nil {
		return false, ConfigTestResult{
			ConfigFile:     configFile,
			ParseSucceeded: false,
			IsValid:        false,
			Error:          err.Error(),
		}, nil
	}

	// Ensure the agent config is valid.
	if err = agentConfig.Validate(); err != nil {
		return false, ConfigTestResult{
			ConfigFile:     configFile,
			ParseSucceeded: true,
			IsValid:        false,
			Error:          err.Error(),
		}, nil
	}

	// All checks passed.
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
