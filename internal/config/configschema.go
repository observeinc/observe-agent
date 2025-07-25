package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"
)

type HostMonitoringLogsConfig struct {
	Enabled                bool     `yaml:"enabled" mapstructure:"enabled"`
	Include                []string `yaml:"include,omitempty" mapstructure:"include"`
	Exclude                []string `yaml:"exclude,omitempty" mapstructure:"exclude"`
	AutoMultilineDetection bool     `yaml:"auto_multiline_detection" mapstructure:"auto_multiline_detection"`
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

type HealthCheckConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled" default:"true"`
	Endpoint string `yaml:"endpoint" mapstructure:"endpoint" default:"localhost:13133"`
	Path     string `yaml:"path" mapstructure:"path" default:"/status"`
}

type ForwardingMetricsConfig struct {
	OutputFormat string `yaml:"output_format,omitempty" mapstructure:"output_format" default:"prometheus" jsonschema:"pattern=^(prometheus|otel)$"`
}

func (config *ForwardingMetricsConfig) OtlpMetrics() bool {
	return config.OutputFormat == "otel"
}

type ForwardingTracesConfig struct {
	MaxSpanDuration string `yaml:"max_span_duration,omitempty" mapstructure:"max_span_duration" default:"1h" jsonschema:"pattern=^(none|[0-9]+(ns|us|µs|ms|s|m|h))$"`
}

type ForwardingConfig struct {
	Enabled bool                    `yaml:"enabled" mapstructure:"enabled" default:"true"`
	Metrics ForwardingMetricsConfig `yaml:"metrics,omitempty" mapstructure:"metrics"`
	Traces  ForwardingTracesConfig  `yaml:"traces,omitempty" mapstructure:"traces"`
}

type InternalTelemetryMetricsConfig struct {
	Enabled bool   `yaml:"enabled" mapstructure:"enabled" default:"true"`
	Host    string `yaml:"host" mapstructure:"host" default:"localhost"`
	Port    int    `yaml:"port" mapstructure:"port" default:"8888"`
	Level   string `yaml:"level" mapstructure:"level" default:"detailed"`
}

type InternalTelemetryLogsConfig struct {
	Enabled  bool   `yaml:"enabled" mapstructure:"enabled" default:"true"`
	Level    string `yaml:"level" mapstructure:"level" default:"${env:OTEL_LOG_LEVEL}"`
	Encoding string `yaml:"encoding" mapstructure:"encoding" default:"console" jsonschema:"pattern=^(console|json)$"`
}

type InternalTelemetryConfig struct {
	Enabled bool                           `yaml:"enabled" mapstructure:"enabled" default:"true"`
	Metrics InternalTelemetryMetricsConfig `yaml:"metrics" mapstructure:"metrics"`
	Logs    InternalTelemetryLogsConfig    `yaml:"logs" mapstructure:"logs"`
}

type REDMetricsConfig struct {
	Enabled bool `yaml:"enabled" mapstructure:"enabled" default:"false"`
}

type ApplicationConfig struct {
	REDMetrics REDMetricsConfig `yaml:"RED_metrics,omitempty" mapstructure:"RED_metrics" json:"RED_metrics"`
}

type AgentConfig struct {
	Token                  string                  `yaml:"token" mapstructure:"token" jsonschema:"required"`
	ObserveURL             string                  `yaml:"observe_url" mapstructure:"observe_url" jsonschema:"required"`
	CloudResourceDetectors []string                `yaml:"cloud_resource_detectors,omitempty" mapstructure:"cloud_resource_detectors"`
	Debug                  bool                    `yaml:"debug,omitempty" mapstructure:"debug"`
	Attributes             map[string]string       `yaml:"attributes,omitempty" mapstructure:"attributes"`
	ResourceAttributes     map[string]string       `yaml:"resource_attributes,omitempty" mapstructure:"resource_attributes"`
	Application            ApplicationConfig       `yaml:"application,omitempty" mapstructure:"application"`
	HealthCheck            HealthCheckConfig       `yaml:"health_check" mapstructure:"health_check"`
	Forwarding             ForwardingConfig        `yaml:"forwarding" mapstructure:"forwarding"`
	InternalTelemetry      InternalTelemetryConfig `yaml:"internal_telemetry" mapstructure:"internal_telemetry"`
	SelfMonitoring         SelfMonitoringConfig    `yaml:"self_monitoring,omitempty" mapstructure:"self_monitoring"`
	HostMonitoring         HostMonitoringConfig    `yaml:"host_monitoring,omitempty" mapstructure:"host_monitoring"`
	OtelConfigOverrides    map[string]any          `yaml:"otel_config_overrides,omitempty" mapstructure:"otel_config_overrides"`
}

func (config *AgentConfig) HasAttributes() bool {
	return len(config.Attributes) > 0
}

func (config *AgentConfig) HasResourceAttributes() bool {
	return len(config.ResourceAttributes) > 0
}

func SetViperDefaults(v *viper.Viper, separator string) {
	var config AgentConfig
	defaults.SetDefaults(&config)
	var confMap map[string]any
	err := mapstructure.Decode(config, &confMap)
	if err != nil {
		panic(err)
	}
	var recursiveDfs func(prefix string, defaults map[string]any)
	recursiveDfs = func(prefix string, defaults map[string]any) {
		for key, val := range defaults {
			if nestedMap, ok := val.(map[string]any); ok {
				// Recurse on nested maps
				recursiveDfs(prefix+key+separator, nestedMap)
			} else {
				// Set this value as default if it's not a map.
				v.SetDefault(prefix+key, val)
			}
		}
	}
	recursiveDfs("", confMap)
}

func AgentConfigFromViper(v *viper.Viper) (*AgentConfig, error) {
	var config AgentConfig
	defaults.SetDefaults(&config)
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

	if config.Forwarding.Metrics.OutputFormat != "prometheus" && config.Forwarding.Metrics.OutputFormat != "otel" {
		return fmt.Errorf("invalid metrics forwarding output format '%s' - valid options are 'prometheus' and 'otel'", config.Forwarding.Metrics.OutputFormat)
	}

	if config.Forwarding.Traces.MaxSpanDuration != "none" {
		if _, err := time.ParseDuration(config.Forwarding.Traces.MaxSpanDuration); err != nil {
			return fmt.Errorf("invalid max span duration '%s' - Expected a number with a valid time unit: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#duration", config.Forwarding.Traces.MaxSpanDuration)
		}
	}

	return nil
}
