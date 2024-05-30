package config

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"

	"observe/agent/cmd/connections"

	"github.com/spf13/viper"
)

func GetAllOtelConfigFilePaths() ([]string, string, error) {
	// Initialize config file paths with base config
	configFilePaths := []string{filepath.Join(GetDefaultConfigFolder(), "otel-collector.yaml")}
	var err error
	// Get additional config paths based on connection configs
	for _, conn := range connections.AllConnectionTypes {
		if viper.IsSet(conn.Name) {
			configFilePaths = append(configFilePaths, conn.GetConfigFilePaths()...)
		}
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
	return configFilePaths, overridePath, nil
}

func SetEnvVars() error {
	collector_url, token := viper.GetString("observe_url"), viper.GetString("token")
	endpoint, err := url.JoinPath(collector_url, "/v2/otel")
	if err != nil {
		return err
	}
	// Setting values from the Observe agent config as env vars to fill in the OTEL collector config
	os.Setenv("OBSERVE_ENDPOINT", endpoint)
	os.Setenv("OBSERVE_TOKEN", "Bearer "+token)
	os.Setenv("FILESTORAGE_PATH", GetDefaultFilestoragePath())
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
		homedir, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		return homedir
	case "windows":
		return "C:\\Program Files\\Observe\\observe-agent\\config"
	case "linux":
		return "/etc/observe-agent"
	default:
		return "/etc/observe-agent"
	}
}

func GetDefaultFilestoragePath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return "~/Library/Application Support/Observe/observe-agent/filestorage"
	case "windows":
		programData := os.Getenv("PROGRAMDATA")
		return path.Join(programData, "Observe", "observe-agent", "filestorage")
	case "linux":
		return "/var/lib/observe-agent/filestorage"
	default:
		return "/var/lib/observe-agent/filestorage"
	}
}
