package diagnose

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const testConfig = `
# Observe data token
token: "some:token"

# Target Observe collection url
observe_url: "https://collect.observeinc.com"

# Debug mode - Sets agent log level to debug
debug: false

# collect metrics and logs pertaining to the agent itself
self_monitoring:
  enabled: true

# collect metrics and logs about the host system
host_monitoring:
  enabled: true
  # collect logs of all running processes from the host system
  logs: 
    enabled: true
  metrics:
    # collect metrics about the host system
    host:
      enabled: true
    # collect metrics about the processes running on the host system
    process:
      enabled: false
`

var (
	validCases = []string{
		testConfig,
		"key:\n  twoSpaces: true\ntoken: some:token\nobserve_url: https://collect.observeinc.com",
	}
	invalidCases = []string{
		// Invalid YAML
		"key:\n\ttabIndented: \"value\"",
		"key:\n  twoSpaces: true\n   threeSpaces: true",
		"\tstartsWithTab: true",
		// Invalid configs
		"",
		"token: some:token\nmissing: URL",
		"missing: token\nobserve_url: https://collect.observeinc.com",
		"token: bad token\nobserve_url: https://collect.observeinc.com",
		"token: some:token\nobserve_url: bad url",
	}
)

func Test_validateAgentConfigYaml(t *testing.T) {
	for _, tc := range validCases {
		err := validateAgentConfigYaml([]byte(tc))
		assert.NoError(t, err)
	}
	for _, tc := range invalidCases {
		err := validateAgentConfigYaml([]byte(tc))
		assert.Error(t, err)
	}
}
