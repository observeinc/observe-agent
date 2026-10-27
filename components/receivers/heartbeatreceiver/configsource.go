package heartbeatreceiver

import (
	"os"
	"sync"

	"go.uber.org/zap"
)

// Configs are handed over in-process rather than via env vars, which child processes inherit and
// which fail execve with E2BIG when over 128 KiB (OB-70021). The env vars remain a legacy fallback.
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
// called before the collector starts; the start command does this.
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
		logger.Error("environment variable is not set", zap.String("env", envVar))
		return ""
	}
	decoded, err := decodeBase64Config(encoded)
	if err != nil {
		logger.Error("failed to decode environment variable from base64", zap.String("env", envVar), zap.Error(err))
		return ""
	}
	return decoded
}
