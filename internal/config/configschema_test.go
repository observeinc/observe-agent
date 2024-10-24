package config

import (
	"testing"

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
