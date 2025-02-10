package connections

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	logger "github.com/observeinc/observe-agent/internal/commands/util"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	OTEL_OVERRIDE_YAML_KEY = "otel_config_overrides"
)

func GetAllOtelConfigFilePaths(ctx context.Context, tmpDir string) ([]string, error) {
	configFilePaths := []string{}
	// If the default otel-collector.yaml exists, add it to the list of config files
	defaultOtelConfigPath := filepath.Join(GetDefaultConfigFolder(), "otel-collector.yaml")
	if _, err := os.Stat(defaultOtelConfigPath); err == nil {
		agentConf, err := config.AgentConfigFromViper(viper.GetViper())
		if err != nil {
			return nil, err
		}
		otelConfigRendered, err := RenderConfigTemplate(ctx, tmpDir, defaultOtelConfigPath, agentConf)
		if err != nil {
			return nil, err
		}
		configFilePaths = append(configFilePaths, otelConfigRendered)
	}
	// Get additional config paths based on connection configs
	for _, conn := range AllConnectionTypes {
		if viper.IsSet(conn.Name) {
			connectionPaths, err := conn.GetConfigFilePaths(ctx, tmpDir)
			if err != nil {
				return nil, err
			}
			configFilePaths = append(configFilePaths, connectionPaths...)
		}
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
			overridePath, err := GetOverrideConfigFile(tmpDir, overrides)
			if err != nil {
				return nil, err
			}
			configFilePaths = append(configFilePaths, overridePath)
		}
	}
	logger.FromCtx(ctx).Debug(fmt.Sprint("Config file paths:", configFilePaths))
	return configFilePaths, nil
}

func SetEnvVars() error {
	collector_url, token, debug := viper.GetString("observe_url"), viper.GetString("token"), viper.GetBool("debug")
	// Ensure the collector url does not end with a slash for consistency. This will allow endpoints to be configured like:
	// "${env:OBSERVE_COLLECTOR_URL}/v1/kubernetes/v1/entity"
	// without worrying about a double slash.
	if collector_url[len(collector_url)-1] == '/' {
		collector_url = collector_url[:len(collector_url)-1]
	}
	otelEndpoint, err := url.JoinPath(collector_url, "/v2/otel")
	if err != nil {
		return err
	}
	promEndpoint, err := url.JoinPath(collector_url, "/v1/prometheus")
	if err != nil {
		return err
	}
	// Setting values from the Observe agent config as env vars to fill in the OTEL collector config
	os.Setenv("OBSERVE_COLLECTOR_URL", collector_url)
	os.Setenv("OBSERVE_OTEL_ENDPOINT", otelEndpoint)
	os.Setenv("OBSERVE_PROMETHEUS_ENDPOINT", promEndpoint)
	os.Setenv("OBSERVE_AUTHORIZATION_HEADER", "Bearer "+token)
	os.Setenv("FILESTORAGE_PATH", GetDefaultFilestoragePath())

	if debug {
		os.Setenv("OTEL_LOG_LEVEL", "DEBUG")
	} else {
		os.Setenv("OTEL_LOG_LEVEL", "INFO")
	}
	return nil
}

func GetOverrideConfigFile(tmpDir string, data map[string]any) (string, error) {
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

func GetDefaultConfigFolder() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return filepath.Join(GetDefaultAgentPath(), "config")
	case "windows":
		return filepath.Join(GetDefaultAgentPath(), "config")
	case "linux":
		return GetDefaultAgentPath()
	default:
		return GetDefaultAgentPath()
	}
}

func GetConfigFragmentFolderPath() string {
	return filepath.Join(GetDefaultAgentPath(), "connections")
}

func GetDefaultAgentPath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return "/usr/local/observe-agent"
	case "windows":
		return os.ExpandEnv("$ProgramFiles\\Observe\\observe-agent")
	case "linux":
		return "/etc/observe-agent"
	default:
		return "/etc/observe-agent"
	}
}

func GetDefaultFilestoragePath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return "/var/lib/observe-agent/filestorage"
	case "windows":
		return os.ExpandEnv("$ProgramData\\Observe\\observe-agent\\filestorage")
	case "linux":
		return "/var/lib/observe-agent/filestorage"
	default:
		return "/var/lib/observe-agent/filestorage"
	}
}
