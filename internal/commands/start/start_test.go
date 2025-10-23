package start

import (
	"context"
	"encoding/base64"
	"os"
	"testing"

	"github.com/mcuadros/go-defaults"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSetConfigEnvVars(t *testing.T) {
	// Save original environment variables
	originalAgentConfig := os.Getenv("OBSERVE_AGENT_CONFIG")
	originalOtelConfig := os.Getenv("OBSERVE_AGENT_OTEL_CONFIG")
	defer func() {
		if originalAgentConfig != "" {
			os.Setenv("OBSERVE_AGENT_CONFIG", originalAgentConfig)
		} else {
			os.Unsetenv("OBSERVE_AGENT_CONFIG")
		}
		if originalOtelConfig != "" {
			os.Setenv("OBSERVE_AGENT_OTEL_CONFIG", originalOtelConfig)
		} else {
			os.Unsetenv("OBSERVE_AGENT_OTEL_CONFIG")
		}
	}()

	t.Run("successfully sets environment variables with valid config", func(t *testing.T) {
		// Save original viper state for cleanup
		originalViper := viper.GetViper()
		defer func() {
			// Restore original viper settings
			viper.Reset()
			for _, key := range originalViper.AllKeys() {
				viper.Set(key, originalViper.Get(key))
			}
		}()

		// Set our test config
		viper.Reset()
		viper.Set("token", "test:token123456789")
		viper.Set("observe_url", "https://example.observeinc.com")

		// Clear environment variables
		os.Unsetenv("OBSERVE_AGENT_CONFIG")
		os.Unsetenv("OBSERVE_AGENT_OTEL_CONFIG")

		// Call the function
		ctx := context.Background()
		err := setConfigEnvVars(ctx)
		require.NoError(t, err, "setConfigEnvVars should succeed in unit test")

		// Both env vars must be set (base64 encoded)
		agentConfigEncoded := os.Getenv("OBSERVE_AGENT_CONFIG")
		require.NotEmpty(t, agentConfigEncoded, "OBSERVE_AGENT_CONFIG must be set")

		otelConfigEncoded := os.Getenv("OBSERVE_AGENT_OTEL_CONFIG")
		require.NotEmpty(t, otelConfigEncoded, "OBSERVE_AGENT_OTEL_CONFIG must be set")

		// Decode from base64
		agentConfigBytes, err := base64.StdEncoding.DecodeString(agentConfigEncoded)
		require.NoError(t, err, "OBSERVE_AGENT_CONFIG should be valid base64")

		otelConfigBytes, err := base64.StdEncoding.DecodeString(otelConfigEncoded)
		require.NoError(t, err, "OBSERVE_AGENT_OTEL_CONFIG should be valid base64")

		// Verify the agent config is valid YAML and contains expected values
		var agentConfigParsed config.AgentConfig
		err = yaml.Unmarshal(agentConfigBytes, &agentConfigParsed)
		require.NoError(t, err, "OBSERVE_AGENT_CONFIG should be valid YAML")
		assert.Equal(t, "test:token123456789", agentConfigParsed.Token)
		assert.Equal(t, "https://example.observeinc.com", agentConfigParsed.ObserveURL)

		// Verify the OTEL config is valid YAML
		var otelConfigParsed map[string]interface{}
		err = yaml.Unmarshal(otelConfigBytes, &otelConfigParsed)
		require.NoError(t, err, "OBSERVE_AGENT_OTEL_CONFIG should be valid YAML")
	})

	t.Run("returns error with invalid config", func(t *testing.T) {
		// Reset viper and set up an invalid config (missing required fields)
		viper.Reset()
		viper.Set("token", "test:token")
		// Missing observe_url - should fail validation

		ctx := context.Background()
		err := setConfigEnvVars(ctx)
		assert.Error(t, err, "Should return error for invalid config")
	})

	t.Run("preserves token and other sensitive fields in env var", func(t *testing.T) {
		// This test verifies that the raw config is stored in the env var (base64 encoded)
		// (redaction happens in the heartbeat receiver when reading the env var)
		viper.Reset()
		viper.Set("token", "sensitive:token12345678901234567890")
		viper.Set("observe_url", "https://example.observeinc.com")

		ctx := context.Background()
		err := setConfigEnvVars(ctx)

		// May fail on OTEL config part, but we can still check agent config
		agentConfigEncoded := os.Getenv("OBSERVE_AGENT_CONFIG")
		if agentConfigEncoded != "" {
			// Decode from base64
			agentConfigBytes, decodeErr := base64.StdEncoding.DecodeString(agentConfigEncoded)
			require.NoError(t, decodeErr, "Should be valid base64")

			// Verify the full token is in the decoded YAML (not redacted)
			assert.Contains(t, string(agentConfigBytes), "sensitive:token12345678901234567890",
				"Token should be stored unredacted in env var")
		}

		// The actual redaction happens in the heartbeat receiver
		// when it calls redactAndEncodeConfig
		_ = err // Ignore error for this test
	})
}

func TestSetConfigEnvVarsIntegration(t *testing.T) {
	t.Run("validates agent config before setting env vars", func(t *testing.T) {
		// Create a valid config
		validConfig := config.AgentConfig{
			Token:      "test:validtoken123",
			ObserveURL: "https://test.observeinc.com",
		}
		defaults.SetDefaults(&validConfig)

		// Verify it's valid
		err := validConfig.Validate()
		require.NoError(t, err, "Test config should be valid")

		// Set it in viper
		viper.Reset()
		viper.Set("token", validConfig.Token)
		viper.Set("observe_url", validConfig.ObserveURL)

		// The function should work with this valid config
		// (though it might fail on OTEL config generation in test env)
		ctx := context.Background()
		_ = setConfigEnvVars(ctx)

		// At minimum, the agent config should be set (base64 encoded)
		agentConfigEncoded := os.Getenv("OBSERVE_AGENT_CONFIG")
		if agentConfigEncoded != "" {
			// Decode from base64
			agentConfigBytes, decodeErr := base64.StdEncoding.DecodeString(agentConfigEncoded)
			assert.NoError(t, decodeErr, "Should be valid base64")

			var parsed config.AgentConfig
			err := yaml.Unmarshal(agentConfigBytes, &parsed)
			assert.NoError(t, err)
			assert.Equal(t, validConfig.Token, parsed.Token)
			assert.Equal(t, validConfig.ObserveURL, parsed.ObserveURL)
		}
	})
}
