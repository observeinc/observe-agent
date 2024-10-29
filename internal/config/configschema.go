package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
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
	Token               string               `yaml:"token"`
	ObserveURL          string               `yaml:"observe_url"`
	SelfMonitoring      SelfMonitoringConfig `yaml:"self_monitoring,omitempty"`
	HostMonitoring      HostMonitoringConfig `yaml:"host_monitoring,omitempty"`
	OtelConfigOverrides map[string]any       `yaml:"otel_config_overrides,omitempty"`
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
