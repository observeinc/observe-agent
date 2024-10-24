package diagnose

import (
	"os"
	"testing"

	"github.com/spf13/viper"
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

func Test_checkConfig(t *testing.T) {
	testCases := []struct {
		confStr    string
		shouldPass bool
	}{
		{testConfig, true},
		{invalidCases[len(invalidCases)-1], false},
	}
	for _, tc := range testCases {
		f, err := os.CreateTemp("", "test-config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(f.Name())
		f.Write([]byte(tc.confStr))

		v := viper.New()
		v.SetConfigFile(f.Name())
		resultAny, err := checkConfig(v)
		assert.NoError(t, err)
		result, ok := resultAny.(ConfigTestResult)
		assert.True(t, ok)
		if tc.shouldPass {
			assert.Empty(t, result.Error)
			assert.True(t, result.Passed)
		} else {
			assert.NotEmpty(t, result.Error)
			assert.False(t, result.Passed)
		}
		assert.Equal(t, f.Name(), result.ConfigFile)
	}
}
