package diagnose

import (
	"embed"
	"fmt"
	"os"

	"github.com/observeinc/observe-agent/internal/root"
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
	if root.CfgFile == "" {
		return nil, fmt.Errorf("no config file defined")
	}
	configFile, err := os.ReadFile(root.CfgFile)
	if err != nil {
		return nil, err
	}
	if err = validateYaml(configFile); err != nil {
		return ConfigTestResult{
			root.CfgFile,
			false,
			err.Error(),
		}, nil
	}
	return ConfigTestResult{
		ConfigFile: root.CfgFile,
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
