package start

import (
	"context"
	"os"
	"testing"

	"github.com/mcuadros/go-defaults"
	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// setupTest isolates viper, the OTel config flags, the heartbeat configs and the legacy env vars.
func setupTest(t *testing.T, token, observeURL string) {
	t.Helper()
	t.Setenv("OBSERVE_AGENT_CONFIG", "")
	t.Setenv("OBSERVE_AGENT_OTEL_CONFIG", "")
	heartbeatreceiver.ResetConfigs()
	setOtelFlags(t)
	viper.Reset()
	viper.Set("token", token)
	viper.Set("observe_url", observeURL)
	t.Cleanup(func() {
		viper.Reset()
		heartbeatreceiver.ResetConfigs()
		setOtelFlags(t)
	})
}

// setOtelFlags registers the OTel config flags on a fresh flag set, which resets them to their
// defaults, and then parses args.
func setOtelFlags(t *testing.T, args ...string) {
	t.Helper()
	fs := pflag.NewFlagSet("test", pflag.ContinueOnError)
	observecol.AddConfigFlags(fs)
	require.NoError(t, fs.Parse(args))
}

func TestSetHeartbeatConfigs(t *testing.T) {
	t.Run("hands configs to the heartbeat receiver without env vars", func(t *testing.T) {
		setupTest(t, "test:token123456789", "https://example.observeinc.com")

		require.NoError(t, setHeartbeatConfigs(context.Background()))

		agentConfigYaml, otelConfigYaml, ok := heartbeatreceiver.Configs()
		require.True(t, ok, "configs must be set")
		require.NotEmpty(t, agentConfigYaml, "agent config must be set")
		require.NotEmpty(t, otelConfigYaml, "OTel config must be set")

		var agentConfigParsed config.AgentConfig
		require.NoError(t, yaml.Unmarshal([]byte(agentConfigYaml), &agentConfigParsed))
		assert.Equal(t, "test:token123456789", agentConfigParsed.Token)
		assert.Equal(t, "https://example.observeinc.com", agentConfigParsed.ObserveURL)

		var otelConfigParsed map[string]interface{}
		require.NoError(t, yaml.Unmarshal([]byte(otelConfigYaml), &otelConfigParsed))
	})

	t.Run("does not set the legacy env vars", func(t *testing.T) {
		setupTest(t, "test:token123456789", "https://example.observeinc.com")

		require.NoError(t, setHeartbeatConfigs(context.Background()))

		// Regression guard (OB-70021): the configs must not be placed in the environment, where
		// child processes such as journalctl would inherit them and hit E2BIG on large configs.
		assert.Empty(t, os.Getenv("OBSERVE_AGENT_CONFIG"))
		assert.Empty(t, os.Getenv("OBSERVE_AGENT_OTEL_CONFIG"))
	})

	t.Run("preserves token unredacted for the receiver", func(t *testing.T) {
		// Redaction happens in the heartbeat receiver, not here.
		setupTest(t, "sensitive:token12345678901234567890", "https://example.observeinc.com")

		require.NoError(t, setHeartbeatConfigs(context.Background()))

		agentConfigYaml, _, ok := heartbeatreceiver.Configs()
		require.True(t, ok)
		assert.Contains(t, agentConfigYaml, "sensitive:token12345678901234567890")
	})

	t.Run("keeps the agent config when OTel rendering fails", func(t *testing.T) {
		setupTest(t, "test:token123456789", "https://example.observeinc.com")
		// A --set value without '=' makes OTel config rendering fail.
		setOtelFlags(t, "--set=invalid")

		require.Error(t, setHeartbeatConfigs(context.Background()))

		agentConfigYaml, otelConfigYaml, ok := heartbeatreceiver.Configs()
		require.True(t, ok, "configs must be set so a partial heartbeat is sent")
		assert.Contains(t, agentConfigYaml, "test:token123456789")
		assert.Empty(t, otelConfigYaml)
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
		setupTest(t, validConfig.Token, validConfig.ObserveURL)

		require.NoError(t, setHeartbeatConfigs(context.Background()))

		agentConfigYaml, _, ok := heartbeatreceiver.Configs()
		require.True(t, ok)
		var parsed config.AgentConfig
		require.NoError(t, yaml.Unmarshal([]byte(agentConfigYaml), &parsed))
		assert.Equal(t, validConfig.Token, parsed.Token)
		assert.Equal(t, validConfig.ObserveURL, parsed.ObserveURL)
	})
}

func TestSetHeartbeatConfigsClearsLegacyEnvVars(t *testing.T) {
	setupTest(t, "test:token123456789", "https://example.observeinc.com")
	t.Setenv("OBSERVE_AGENT_CONFIG", "aW5oZXJpdGVk")
	t.Setenv("OBSERVE_AGENT_OTEL_CONFIG", "aW5oZXJpdGVk")

	require.NoError(t, setHeartbeatConfigs(context.Background()))

	_, found := os.LookupEnv("OBSERVE_AGENT_CONFIG")
	assert.False(t, found)
	_, found = os.LookupEnv("OBSERVE_AGENT_OTEL_CONFIG")
	assert.False(t, found)
}

func TestSetHeartbeatConfigsAgentConfigFailure(t *testing.T) {
	setupTest(t, "test:token123456789", "https://example.observeinc.com")
	// A map cannot be decoded into the string token field, so AgentConfigFromViper fails.
	viper.Set("token", map[string]int{"a": 1})

	require.Error(t, setHeartbeatConfigs(context.Background()))

	agentConfigYaml, otelConfigYaml, ok := heartbeatreceiver.Configs()
	require.True(t, ok, "configs must be marked set so the env var fallback is skipped")
	assert.Empty(t, agentConfigYaml)
	assert.Empty(t, otelConfigYaml)
}
