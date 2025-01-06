package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/viper"
)

type HostMonitoringLogsConfig struct {
	Enabled bool     `yaml:"enabled" mapstructure:"enabled"`
	Include []string `yaml:"include,omitempty" mapstructure:"include"`
	Exclude []string `yaml:"exclude,omitempty" mapstructure:"exclude"`
}

type HostMonitoringHostMetricsConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type HostMonitoringProcessMetricsConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type HostMonitoringMetricsConfig struct {
	Host    HostMonitoringHostMetricsConfig    `yaml:"host,omitempty" mapstructure:"host"`
	Process HostMonitoringProcessMetricsConfig `yaml:"process,omitempty" mapstructure:"process"`
}

type HostMonitoringConfig struct {
	Enabled bool                        `yaml:"enabled" mapstructure:"enabled"`
	Logs    HostMonitoringLogsConfig    `yaml:"logs,omitempty" mapstructure:"logs"`
	Metrics HostMonitoringMetricsConfig `yaml:"metrics,omitempty" mapstructure:"metrics"`
}

type SelfMonitoringConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled"`
}

type AgentConfig struct {
	Token                  string               `yaml:"token" mapstructure:"token"`
	ObserveURL             string               `yaml:"observe_url" mapstructure:"observe_url"`
	CloudResourceDetectors []string             `yaml:"cloud_resource_detectors,omitempty" mapstructure:"cloud_resource_detectors"`
	SelfMonitoring         SelfMonitoringConfig `yaml:"self_monitoring,omitempty" mapstructure:"self_monitoring"`
	HostMonitoring         HostMonitoringConfig `yaml:"host_monitoring,omitempty" mapstructure:"host_monitoring"`
	OtelConfigOverrides    map[string]any       `yaml:"otel_config_overrides,omitempty" mapstructure:"otel_config_overrides"`
}

func AgentConfigFromViper(v *viper.Viper) (*AgentConfig, error) {
	var config AgentConfig
	err := v.Unmarshal(&config)
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
