package diagnose

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

type ConfigTestResult struct {
	ConfigFile string
	Passed     bool
	Error      string
}

func validateYaml(yamlContent []byte) error {
	m := make(map[string]any)
	return yaml.Unmarshal(yamlContent, &m)
}

func checkConfig() (any, error) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return nil, fmt.Errorf("no config file defined")
	}
	contents, err := os.ReadFile(configFile)
	if err != nil {
		return nil, err
	}
	if err = validateYaml(contents); err != nil {
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
