package initconfig

import (
	"os"
	"regexp"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/observeinc/observe-agent/internal/config"
	"gopkg.in/yaml.v3"
)

const otelOverrideSection = `
# otel_config_overrides:
#   exporters:
#     debug:
#       verbosity: detailed
#       sampling_initial: 5
#       sampling_thereafter: 200
#   service:
#     pipelines:
#       metrics:
#         receivers: [hostmetrics]
#         processors: [memory_limiter]
#         exporters: [debug]
`

var sectionComments map[string]string = map[string]string{
	"token":       "Observe data token",
	"observe_url": "Target Observe collection url",
	"debug":       "Debug mode - Sets agent log level to debug",
	"environment": "test env comment",
}

func writeConfigFile(f *os.File, agentConfig *config.AgentConfig, includeAllOptions bool) error {
	var yamlBytes []byte
	var err error
	if includeAllOptions {
		var agentConfigMap map[string]any
		err = mapstructure.Decode(agentConfig, &agentConfigMap)
		if err != nil {
			return err
		}
		yamlBytes, err = yaml.Marshal(&agentConfigMap)
		if err != nil {
			return err
		}
	} else {
		yamlBytes, err = yaml.Marshal(agentConfig)
		if err != nil {
			return err
		}
	}
	yamlStr := string(yamlBytes)

	// Add empty lines between top level sections.
	yamlStr = regexp.MustCompile(`(?m)^(\w+:.*)$`).ReplaceAllString(yamlStr, "\n$1")

	// Add comments before yaml keys.
	for commentKey, commentValue := range sectionComments {
		yamlStr = regexp.MustCompile(`(?m)^([`+" \t"+`]*)`+commentKey+`:.*$`).ReplaceAllString(yamlStr, "$1# "+commentValue+"\n$0")
	}

	// Clean up whitespace
	yamlStr = strings.Trim(yamlStr, " \n\t\r") + "\n"

	// Add the otel config overrides comment if there is no section present.
	if !strings.Contains(yamlStr, "otel_config_overrides:") {
		yamlStr += "\n" + strings.Trim(otelOverrideSection, " \n\t\r") + "\n"
	}

	_, err = f.WriteString(yamlStr)
	return err
}
