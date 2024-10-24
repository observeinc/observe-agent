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
	ConfigFile string
	Passed     bool
	Error      string
}

func validateAgentConfigYaml(yamlContent []byte) error {
	var conf config.AgentConfig
	err := yaml.Unmarshal(yamlContent, &conf)
	if err != nil {
		return err
	}
	return conf.Validate()
}

func checkConfig(v *viper.Viper) (any, error) {
	configFile := v.ConfigFileUsed()
	if configFile == "" {
		return nil, fmt.Errorf("no config file defined")
	}
	contents, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	if err = validateAgentConfigYaml(contents); err != nil {
		return ConfigTestResult{
			configFile,
			false,
			err.Error(),
		}, nil
	}
	return ConfigTestResult{
		ConfigFile: configFile,
		Passed:     true,
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
