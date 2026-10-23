package heartbeatreceiver

import (
	"os"
	"sync"
)

// The config heartbeat needs the resolved observe-agent and OTel configs. These used to be passed
// only via the OBSERVE_AGENT_CONFIG / OBSERVE_AGENT_OTEL_CONFIG env vars, but every child process
// (e.g. journalctl spawned by the journald receiver) inherits the environment, and Linux rejects
// any single env string over MAX_ARG_STRLEN (128 KiB) with E2BIG. Large configs therefore broke
// any receiver that execs a child. The configs are now handed over in-process instead.

var (
	configsMu       sync.RWMutex
	configsSet      bool
	agentConfigYaml string
	otelConfigYaml  string
)

// SetConfigs stores the raw (not base64 encoded) observe-agent and OTel config YAML for the
// config heartbeat. It must be called before the collector starts.
func SetConfigs(agentConfig, otelConfig string) {
	configsMu.Lock()
	defer configsMu.Unlock()
	agentConfigYaml = agentConfig
	otelConfigYaml = otelConfig
	configsSet = true
}

// resetConfigs clears in-process configs; used by tests.
func resetConfigs() {
	configsMu.Lock()
	defer configsMu.Unlock()
	agentConfigYaml, otelConfigYaml, configsSet = "", "", false
}

// getConfigs returns the configs set via SetConfigs. If SetConfigs was never called it falls back
// to the legacy base64 encoded env vars, returning their encoded values and fromEnv=true.
func getConfigs() (agentConfig, otelConfig string, fromEnv bool) {
	configsMu.RLock()
	defer configsMu.RUnlock()
	if configsSet {
		return agentConfigYaml, otelConfigYaml, false
	}
	return os.Getenv("OBSERVE_AGENT_CONFIG"), os.Getenv("OBSERVE_AGENT_OTEL_CONFIG"), true
}

// Configs returns the configs set via SetConfigs.
func Configs() (agentConfig, otelConfig string) {
	configsMu.RLock()
	defer configsMu.RUnlock()
	return agentConfigYaml, otelConfigYaml
}
