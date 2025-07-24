package config

import (
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

}

func TestAgentConfigFromViper(t *testing.T) {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	v.Set("token", "some:token")
	v.Set("observe_url", "https://observeinc.com")
	v.Set("host_monitoring::enabled", true)
	config, err := AgentConfigFromViper(v)
	assert.NoError(t, err)
	assert.Equal(t, "some:token", config.Token)
	assert.Equal(t, "https://observeinc.com", config.ObserveURL)
	assert.Equal(t, true, config.HostMonitoring.Enabled)

	// Validate that defaults are set when the value is not in the viper config
	assert.Equal(t, true, config.HealthCheck.Enabled)
	assert.Equal(t, true, config.Forwarding.Enabled)

	// Validate that defaults are overridden by present values
	v.Set("health_check::enabled", false)
	config, err = AgentConfigFromViper(v)
	assert.NoError(t, err)
	assert.Equal(t, false, config.HealthCheck.Enabled)
	assert.Equal(t, true, config.Forwarding.Enabled)
}
