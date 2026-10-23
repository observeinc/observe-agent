package heartbeatreceiver

import (
	"os"
	"sync"

	"go.uber.org/zap"
)

// The config heartbeat needs the resolved observe-agent and OTel configs. These used to be passed
// only via the OBSERVE_AGENT_CONFIG / OBSERVE_AGENT_OTEL_CONFIG env vars, but every child process
// (e.g. journalctl spawned by the journald receiver) inherits the environment, and Linux rejects
// any single env string over MAX_ARG_STRLEN (128 KiB) with E2BIG. Large configs therefore broke
// any receiver that execs a child. The configs are now handed over in-process instead.

const (
	agentConfigEnvVar = "OBSERVE_AGENT_CONFIG"
	otelConfigEnvVar  = "OBSERVE_AGENT_OTEL_CONFIG"
)

var (
	configsMu       sync.RWMutex
	configsSet      bool
	agentConfigYaml string
	otelConfigYaml  string
)

// SetConfigs stores the raw (not base64 encoded) observe-agent and OTel config YAML for the
// config heartbeat. Either may be empty, in which case a partial heartbeat is sent. It must be
// called before the collector starts.
func SetConfigs(agentConfig, otelConfig string) {
	configsMu.Lock()
	defer configsMu.Unlock()
	agentConfigYaml = agentConfig
	otelConfigYaml = otelConfig
	configsSet = true
}

// Configs returns the configs stored by SetConfigs. ok is false if SetConfigs has not been called.
func Configs() (agentConfig, otelConfig string, ok bool) {
	configsMu.RLock()
	defer configsMu.RUnlock()
	return agentConfigYaml, otelConfigYaml, configsSet
}

// ResetConfigs clears the configs stored by SetConfigs. Intended for tests.
func ResetConfigs() {
	configsMu.Lock()
	defer configsMu.Unlock()
	agentConfigYaml, otelConfigYaml, configsSet = "", "", false
}

// getConfigs returns the raw config YAML for the heartbeat. It prefers configs stored by
// SetConfigs and otherwise falls back to the legacy base64 encoded env vars. Missing or
// undecodable configs are logged and returned as empty strings.
func getConfigs(logger *zap.Logger) (agentConfig, otelConfig string) {
	if agentConfig, otelConfig, ok := Configs(); ok {
		if agentConfig == "" {
			logger.Error("observe-agent config was not provided")
		}
		if otelConfig == "" {
			logger.Error("OTel config was not provided")
		}
		return agentConfig, otelConfig
	}
	return getEnvConfig(logger, agentConfigEnvVar), getEnvConfig(logger, otelConfigEnvVar)
}

func getEnvConfig(logger *zap.Logger, envVar string) string {
	encoded := os.Getenv(envVar)
	if encoded == "" {
		logger.Error(envVar + " environment variable is not set")
		return ""
	}
	decoded, err := decodeBase64Config(encoded)
	if err != nil {
		logger.Error("failed to decode "+envVar+" from base64", zap.Error(err))
		return ""
	}
	return decoded
}
