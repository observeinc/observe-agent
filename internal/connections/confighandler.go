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
)

func GetAllOtelConfigFilePaths(ctx context.Context, tmpDir string) ([]string, string, error) {
	configFilePaths := []string{}
	// If the default otel-collector.yaml exists, add it to the list of config files
	defaultOtelConfigPath := filepath.Join(GetDefaultConfigFolder(), "otel-collector.yaml")
	if _, err := os.Stat(defaultOtelConfigPath); err == nil {
		agentConf, err := config.AgentConfigFromViper(viper.GetViper())
		if err != nil {
			return nil, "", err
		}
		otelConfigRendered, err := RenderConfigTemplate(ctx, tmpDir, defaultOtelConfigPath, agentConf)
		if err != nil {
			return nil, "", err
		}
		configFilePaths = append(configFilePaths, otelConfigRendered)
	}
	var err error
	// Get additional config paths based on connection configs
	for _, conn := range AllConnectionTypes {
		if viper.IsSet(conn.Name) {
			connectionPaths, err := conn.GetConfigFilePaths(ctx, tmpDir)
			if err != nil {
				return nil, "", err
			}
			configFilePaths = append(configFilePaths, connectionPaths...)
		}
	}
	// Read in otel-config flag and add to paths if set
	if viper.IsSet("otelConfigFile") {
		configFilePaths = append(configFilePaths, viper.GetString("otelConfigFile"))
	}
	// Generate override file and include path if overrides provided
	var overridePath string
	if viper.IsSet("otel_config_overrides") {
		overridePath, err = GetOverrideConfigFile(viper.Sub("otel_config_overrides"))
		if err != nil {
			return configFilePaths, overridePath, err
		}
		configFilePaths = append(configFilePaths, overridePath)
	}
	logger.FromCtx(ctx).Info(fmt.Sprint("Config file paths:", configFilePaths))
	return configFilePaths, overridePath, nil
}

func SetEnvVars() error {
	collector_url, token, debug := viper.GetString("observe_url"), viper.GetString("token"), viper.GetBool("debug")
	endpoint, err := url.JoinPath(collector_url, "/v2/otel")
	if err != nil {
		return err
	}
	// Setting values from the Observe agent config as env vars to fill in the OTEL collector config
	os.Setenv("OBSERVE_ENDPOINT", endpoint)
	os.Setenv("OBSERVE_TOKEN", "Bearer "+token)
	os.Setenv("FILESTORAGE_PATH", GetDefaultFilestoragePath())

	if debug {
		os.Setenv("OTEL_LOG_LEVEL", "DEBUG")
	} else {
		os.Setenv("OTEL_LOG_LEVEL", "INFO")
	}
	return nil
}

func GetOverrideConfigFile(sub *viper.Viper) (string, error) {
	f, err := os.CreateTemp("", "otel-config-overrides-*.yaml")
	if err != nil {
		return "", fmt.Errorf("failed to create config file to write to: %w", err)
	}
	err = sub.WriteConfigAs(f.Name())
	if err != nil {
		return f.Name(), fmt.Errorf("failed to write otel config overrides to file: %w", err)
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
