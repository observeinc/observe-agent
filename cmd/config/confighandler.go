package config

import (
	"fmt"
	"net/url"
	"os"
	"runtime"

	"observe/agent/cmd/connections"

	"github.com/spf13/viper"
)

const BaseOtelCollectorConfigFilePath = "/etc/observe-agent/otel-collector.yaml"

func GetAllOtelConfigFilePaths() ([]string, string, error) {
	// Initialize config file paths with base config
	configFilePaths := []string{BaseOtelCollectorConfigFilePath}
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
	fsPath := viper.GetString("filestorage_path")
	// Setting values from the Observe agent config as env vars to fill in the OTEL collector config
	os.Setenv("OBSERVE_ENDPOINT", endpoint)
	os.Setenv("OBSERVE_TOKEN", "Bearer "+token)
	if fsPath != "" {
		os.Setenv("FILESTORAGE_PATH", fsPath)
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
	switch os := runtime.GOOS; os {
	case "darwin":
		return "$HOME"
	case "windows":
		return "%PROGRAMDATA%\\observe-agent\\config"
	case "linux":
		return "/etc/observe-agent"
	default:
		return "/etc/observe-agent"
	}
}
