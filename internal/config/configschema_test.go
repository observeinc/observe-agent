package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAgentConfigValidate(t *testing.T) {
	validConfig := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
	}
	assert.NoError(t, validConfig.Validate())

	missingURLConfig := AgentConfig{
		Token:      "some:token",
		ObserveURL: "",
	}
	assert.ErrorContains(t, missingURLConfig.Validate(), "missing ObserveURL")

	invalidURLConfig1 := AgentConfig{
		Token:      "some:token",
		ObserveURL: "observeinc.com",
	}
	assert.ErrorContains(t, invalidURLConfig1.Validate(), "missing scheme for ObserveURL")

	invalidURLConfig2 := AgentConfig{
		Token:      "some:token",
		ObserveURL: "http://",
	}
	assert.ErrorContains(t, invalidURLConfig2.Validate(), "missing host for ObserveURL")

	missingTokenConfig := AgentConfig{
		Token:      "",
		ObserveURL: "https://observeinc.com",
	}
	assert.ErrorContains(t, missingTokenConfig.Validate(), "missing Token")

	invalidTokenConfig := AgentConfig{
		Token:      "1234",
		ObserveURL: "https://observeinc.com",
	}
	assert.ErrorContains(t, invalidTokenConfig.Validate(), "invalid Token")
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
