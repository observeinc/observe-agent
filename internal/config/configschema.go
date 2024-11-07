package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type HostMonitoringLogsConfig struct {
	Enabled bool     `yaml:"enabled"`
	Include []string `yaml:"include,omitempty"`
}

type HostMonitoringHostMetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type HostMonitoringProcessMetricsConfig struct {
	Enabled bool `yaml:"enabled"`
}

type HostMonitoringMetricsConfig struct {
	Host    HostMonitoringHostMetricsConfig    `yaml:"host,omitempty"`
	Process HostMonitoringProcessMetricsConfig `yaml:"process,omitempty"`
}

type HostMonitoringConfig struct {
	Enabled bool                        `yaml:"enabled"`
	Logs    HostMonitoringLogsConfig    `yaml:"logs,omitempty"`
	Metrics HostMonitoringMetricsConfig `yaml:"metrics,omitempty"`
}

type SelfMonitoringConfig struct {
	Enabled bool `yaml:"enabled"`
}

type AgentConfig struct {
	Token                  string               `yaml:"token"`
	ObserveURL             string               `yaml:"observe_url"`
	CloudResourceDetectors []string             `yaml:"cloud_resource_detectors,omitempty"`
	SelfMonitoring         SelfMonitoringConfig `yaml:"self_monitoring,omitempty"`
	HostMonitoring         HostMonitoringConfig `yaml:"host_monitoring,omitempty"`
	OtelConfigOverrides    map[string]any       `yaml:"otel_config_overrides,omitempty"`
}

func UnmarshalViperThroughYaml(v *viper.Viper, out any) error {
	// First unmarshal viper into a map
	var viperConfig map[string]any
	if err := viper.Unmarshal(&viperConfig); err != nil {
		return err
	}
	// Next convert the map into yaml bytes
	viperConfigYaml, err := yaml.Marshal(viperConfig)
	if err != nil {
		return err
	}
	// Finally unmarshal the yaml bytes into the out struct
	return yaml.Unmarshal(viperConfigYaml, out)
}

func AgentConfigFromViper(v *viper.Viper) (*AgentConfig, error) {
	var config AgentConfig
	err := UnmarshalViperThroughYaml(v, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func (config *AgentConfig) Validate() error {
	if config.ObserveURL == "" {
		return errors.New("missing ObserveURL")
	}
	u, err := url.Parse(config.ObserveURL)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		return fmt.Errorf("missing scheme for ObserveURL %s", config.ObserveURL)
	}
	if u.Host == "" {
		return fmt.Errorf("missing host for ObserveURL %s", config.ObserveURL)
	}

	if config.Token == "" {
		return errors.New("missing Token")
	}
	if !strings.Contains(config.Token, ":") {
		return errors.New("invalid Token, the provided value may be the token ID instead of the token itself")
	}
	return nil
}
