package start

import (
	"context"
	"os"
	"testing"

	"github.com/mcuadros/go-defaults"
	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestSetHeartbeatConfigs(t *testing.T) {
	t.Run("hands configs to the heartbeat receiver without env vars", func(t *testing.T) {
		originalViper := viper.GetViper()
		defer func() {
			viper.Reset()
			for _, key := range originalViper.AllKeys() {
				viper.Set(key, originalViper.Get(key))
			}
		}()

		viper.Reset()
		viper.Set("token", "test:token123456789")
		viper.Set("observe_url", "https://example.observeinc.com")

		os.Unsetenv("OBSERVE_AGENT_CONFIG")
		os.Unsetenv("OBSERVE_AGENT_OTEL_CONFIG")

		err := setHeartbeatConfigs(context.Background())
		require.NoError(t, err, "setHeartbeatConfigs should succeed in unit test")

		// Regression guard (OB-70021): the configs must not be placed in the environment, where
		// child processes such as journalctl would inherit them and hit E2BIG on large configs.
		assert.Empty(t, os.Getenv("OBSERVE_AGENT_CONFIG"))
		assert.Empty(t, os.Getenv("OBSERVE_AGENT_OTEL_CONFIG"))

		agentConfigYaml, otelConfigYaml := heartbeatreceiver.Configs()
		require.NotEmpty(t, agentConfigYaml, "agent config must be set")
		require.NotEmpty(t, otelConfigYaml, "OTel config must be set")

		var agentConfigParsed config.AgentConfig
		require.NoError(t, yaml.Unmarshal([]byte(agentConfigYaml), &agentConfigParsed))
		assert.Equal(t, "test:token123456789", agentConfigParsed.Token)
		assert.Equal(t, "https://example.observeinc.com", agentConfigParsed.ObserveURL)

		var otelConfigParsed map[string]interface{}
		require.NoError(t, yaml.Unmarshal([]byte(otelConfigYaml), &otelConfigParsed))
	})

	t.Run("preserves token unredacted for the receiver", func(t *testing.T) {
		// Redaction happens in the heartbeat receiver, not here.
		viper.Reset()
		viper.Set("token", "sensitive:token12345678901234567890")
		viper.Set("observe_url", "https://example.observeinc.com")

		_ = setHeartbeatConfigs(context.Background())

		agentConfigYaml, _ := heartbeatreceiver.Configs()
		assert.Contains(t, agentConfigYaml, "sensitive:token12345678901234567890")
	})
}

func TestSetHeartbeatConfigsIntegration(t *testing.T) {
	t.Run("validates agent config before setting configs", func(t *testing.T) {
		validConfig := config.AgentConfig{
			Token:      "test:validtoken123",
			ObserveURL: "https://test.observeinc.com",
		}
		defaults.SetDefaults(&validConfig)
		require.NoError(t, validConfig.Validate(), "Test config should be valid")

		viper.Reset()
		viper.Set("token", validConfig.Token)
		viper.Set("observe_url", validConfig.ObserveURL)

		_ = setHeartbeatConfigs(context.Background())

		agentConfigYaml, _ := heartbeatreceiver.Configs()
		var parsed config.AgentConfig
		require.NoError(t, yaml.Unmarshal([]byte(agentConfigYaml), &parsed))
		assert.Equal(t, validConfig.Token, parsed.Token)
		assert.Equal(t, validConfig.ObserveURL, parsed.ObserveURL)
	})
}
