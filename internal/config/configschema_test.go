package config

import (
	"os"
	"testing"

	"github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAgentConfigValidate(t *testing.T) {
	validConfig := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
	}
	defaults.SetDefaults(&validConfig)
	assert.NoError(t, validConfig.Validate())

	validConfigOtel := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
		Forwarding: ForwardingConfig{
			Metrics: ForwardingMetricsConfig{
				OutputFormat: "otel",
			},
		},
	}
	defaults.SetDefaults(&validConfigOtel)
	assert.NoError(t, validConfigOtel.Validate())

	missingURLConfig := AgentConfig{
		Token:      "some:token",
		ObserveURL: "",
	}
	defaults.SetDefaults(&missingURLConfig)
	assert.ErrorContains(t, missingURLConfig.Validate(), "missing ObserveURL")

	invalidURLConfig1 := AgentConfig{
		Token:      "some:token",
		ObserveURL: "observeinc.com",
	}
	defaults.SetDefaults(&invalidURLConfig1)
	assert.ErrorContains(t, invalidURLConfig1.Validate(), "missing scheme for ObserveURL")

	invalidURLConfig2 := AgentConfig{
		Token:      "some:token",
		ObserveURL: "http://",
	}
	defaults.SetDefaults(&invalidURLConfig2)
	assert.ErrorContains(t, invalidURLConfig2.Validate(), "missing host for ObserveURL")

	missingTokenConfig := AgentConfig{
		Token:      "",
		ObserveURL: "https://observeinc.com",
	}
	defaults.SetDefaults(&missingTokenConfig)
	assert.ErrorContains(t, missingTokenConfig.Validate(), "missing Token")

	invalidTokenConfig := AgentConfig{
		Token:      "1234",
		ObserveURL: "https://observeinc.com",
	}
	defaults.SetDefaults(&invalidTokenConfig)
	assert.ErrorContains(t, invalidTokenConfig.Validate(), "invalid Token")

	invalidMetricsForwardingFormat := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
		Forwarding: ForwardingConfig{
			Enabled: true,
			Metrics: ForwardingMetricsConfig{
				OutputFormat: "invalid",
			},
		},
	}
	defaults.SetDefaults(&invalidMetricsForwardingFormat)
	assert.ErrorContains(t, invalidMetricsForwardingFormat.Validate(), "invalid metrics forwarding output format")

	emptyMetricsForwardingFormat := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
	}
	defaults.SetDefaults(&emptyMetricsForwardingFormat)
	emptyMetricsForwardingFormat.Forwarding.Metrics.OutputFormat = ""
	assert.ErrorContains(t, emptyMetricsForwardingFormat.Validate(), "invalid metrics forwarding output format")

	invalidMaxSpanDurationFormat := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
		Forwarding: ForwardingConfig{
			Enabled: true,
			Traces: ForwardingTracesConfig{
				MaxSpanDuration: "five months",
			},
		},
	}
	defaults.SetDefaults(&invalidMaxSpanDurationFormat)
	assert.ErrorContains(t, invalidMaxSpanDurationFormat.Validate(), "invalid max span duration 'five months' - Expected a number with a valid time unit: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#duration")

	os.Setenv("OBSERVE_AGENT_TEST_ID", "123")
	testEnvVarConfig := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://${env:OBSERVE_AGENT_TEST_ID}.collect.observeinc.com",
	}
	defaults.SetDefaults(&testEnvVarConfig)
	assert.NoError(t, testEnvVarConfig.Validate())
}

func TestAgentConfigFromViper(t *testing.T) {
	config, err := AgentConfigFromViper(nil)
	assert.Error(t, err, "no viper instance provided")
	assert.Nil(t, config)

	type viperCase struct {
		name       string
		options    map[string]any
		configMode string
		validate   func(t *testing.T, config *AgentConfig)
	}

	cases := []viperCase{
		{
			name: "set values stick and unset values fall back to defaults",
			options: map[string]any{
				"token":                    "some:token",
				"observe_url":              "https://observeinc.com",
				"host_monitoring::enabled": true,
				"resource_attributes":      map[string]string{"deployment.environment.name": "test"},
			},
			configMode: "linux",
			validate: func(t *testing.T, config *AgentConfig) {
				assert.Equal(t, "some:token", config.Token)
				assert.Equal(t, "https://observeinc.com", config.ObserveURL)
				assert.Equal(t, true, config.HostMonitoring.Enabled)
				assert.Equal(t, "test", config.ResourceAttributes["deployment.environment.name"])

				// Defaults should be present when the value is not in the viper config.
				assert.Equal(t, true, config.HealthCheck.Enabled)
				assert.Equal(t, true, config.Forwarding.Enabled)
			},
		},
		{
			name: "user-supplied RED_metrics resource_dimensions in docker mode are not augmented with container.id",
			options: map[string]any{
				"token":                             "some:token",
				"observe_url":                       "https://observeinc.com",
				"application::RED_metrics::enabled": true,
				"application::RED_metrics::resource_dimensions": []string{"service.namespace", "service.version"},
			},
			configMode: "docker",
			validate: func(t *testing.T, config *AgentConfig) {
				assert.Equal(t, true, config.Application.REDMetrics.Enabled)
				assert.Equal(t, []string{"service.namespace", "service.version"}, config.Application.REDMetrics.ResourceDimensions)
				assert.NotContains(t, config.Application.REDMetrics.ResourceDimensions, "container.id")
				assert.NotContains(t, config.Application.REDMetrics.ResourceDimensions, "host.name")

				// OnlyGenerateForAPMSpans should default to true.
				assert.Equal(t, true, config.Application.REDMetrics.OnlyGenerateForAPMSpans)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.NewWithOptions(viper.KeyDelimiter("::"))
			SetViperDefaults(v, "::", tc.configMode)
			for k, val := range tc.options {
				v.Set(k, val)
			}
			config, err := AgentConfigFromViper(v)
			assert.NoError(t, err)
			tc.validate(t, config)
		})
	}
}

func TestSetViperDefaultsDockerConfigMode(t *testing.T) {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	SetViperDefaults(v, "::", "docker")

	assert.Equal(t, "0.0.0.0:4318", v.GetString("forwarding::endpoints::http"))
	assert.Equal(t, "0.0.0.0:4317", v.GetString("forwarding::endpoints::grpc"))
	assert.Equal(t, "0.0.0.0:13133", v.GetString("health_check::endpoint"))
	assert.Equal(t, "0.0.0.0", v.GetString("internal_telemetry::metrics::host"))
	assert.Contains(t, v.GetStringSlice("application::RED_metrics::resource_dimensions"), "container.id")
	assert.Contains(t, v.GetStringSlice("application::RED_metrics::resource_dimensions"), "host.name")
}

func TestSetViperDefaultsNonDockerConfigMode(t *testing.T) {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	SetViperDefaults(v, "::", "linux")

	assert.Equal(t, "localhost:4318", v.GetString("forwarding::endpoints::http"))
	assert.Equal(t, "localhost:4317", v.GetString("forwarding::endpoints::grpc"))
	assert.Equal(t, "localhost:13133", v.GetString("health_check::endpoint"))
	assert.Equal(t, "localhost", v.GetString("internal_telemetry::metrics::host"))
	assert.Contains(t, v.GetStringSlice("application::RED_metrics::resource_dimensions"), "host.name")
	assert.NotContains(t, v.GetStringSlice("application::RED_metrics::resource_dimensions"), "container.id")
}
