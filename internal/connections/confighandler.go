package connections

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/utils"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	OTEL_OVERRIDE_YAML_KEY = "otel_config_overrides"
)

func SetupAndGetConfigFiles(ctx context.Context) ([]string, func(), error) {
	// Set up our temp dir and temp config files
	tmpDir, err := os.MkdirTemp("", TempFilesFolder)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}
	configFilePaths, err := getAllOtelConfigFilePaths(ctx, tmpDir)
	if err != nil {
		cleanup()
		return nil, nil, err
	}
	return configFilePaths, cleanup, nil
}

func getAllOtelConfigFilePaths(ctx context.Context, tmpDir string) ([]string, error) {
	configFilePaths := []string{}
	// Get additional config paths based on connection configs
	agentConfig, err := config.AgentConfigFromViper(viper.GetViper())
	if err != nil {
		return nil, err
	}
	for _, conn := range AllConnectionTypes {
		connectionPaths, err := conn.GetBundledConfigs(ctx, tmpDir, agentConfig)
		if err != nil {
			return nil, err
		}
		configFilePaths = append(configFilePaths, connectionPaths...)
	}
	// Generate override file and include path if overrides provided
	if viper.IsSet(OTEL_OVERRIDE_YAML_KEY) {
		// GetStringMap is more lenient with respect to conversions than Sub, which only handles maps.
		overrides := viper.GetStringMap(OTEL_OVERRIDE_YAML_KEY)
		if len(overrides) == 0 {
			stringData := viper.GetString(OTEL_OVERRIDE_YAML_KEY)
			// If this was truly set to empty, then ignore it.
			if stringData != "" {
				// Viper can handle overrides set in the agent config, or passed in as an env var as a JSON string.
				// For consistency, we also want to accept an env var as a YAML string.
				err := yaml.Unmarshal([]byte(stringData), &overrides)
				if err != nil {
					return nil, fmt.Errorf("%s was provided but could not be parsed", OTEL_OVERRIDE_YAML_KEY)
				}
			}
		}
		// Only create the config file if there are overrides present (ie ignore empty maps)
		if len(overrides) != 0 {
			overridePath, err := getOverrideConfigFile(tmpDir, overrides)
			if err != nil {
				return nil, err
			}
			configFilePaths = append(configFilePaths, overridePath)
		}
	}
	logger.FromCtx(ctx).Debug(fmt.Sprint("Config file paths:", configFilePaths))
	return configFilePaths, nil
}

func getOverrideConfigFile(tmpDir string, data map[string]any) (string, error) {
	f, err := os.CreateTemp(tmpDir, "otel-config-overrides-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create config file to write to: %w", err)
	}
	contents, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal otel config overrides: %w", err)
	}
	_, err = f.Write([]byte(contents))
	if err != nil {
		return "", fmt.Errorf("failed to write otel config overrides to file: %w", err)
	}
	return f.Name(), nil
}

func GetConfigFragmentFolderPath() string {
	return filepath.Join(utils.GetDefaultAgentPath(), "connections")
}

func GetDefaultAgentPath() string {
	return utils.GetDefaultAgentPath()
}
