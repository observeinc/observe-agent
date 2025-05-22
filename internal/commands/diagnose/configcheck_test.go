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

forwarding:
  enabled: true
  metrics:
    output_format: otel

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

var testCases = []struct {
	confStr     string
	shouldParse bool
	isValid     bool
}{
	// Invalid YAML
	{"key:\n\ttabIndented: \"value\"", false, false},
	{"key:\n  twoSpaces: true\n   threeSpaces: true", false, false},
	{"\tstartsWithTab: true", false, false},
	// Invalid Configs
	{"", true, false},
	{"token: some:token\nmissing: URL", true, false},
	{"missing: token\nobserve_url: https://collect.observeinc.com", true, false},
	{"token: bad token\nobserve_url: https://collect.observeinc.com", true, false},
	{"token: some:token\nobserve_url: bad url", true, false},
	// Valid configs
	{testConfig, true, true},
	{"key:\n  twoSpaces: true\ntoken: some:token\nobserve_url: https://collect.observeinc.com", true, true},
}

func Test_checkConfig(t *testing.T) {
	for _, tc := range testCases {
		f, err := os.CreateTemp("", "test-config-*.yaml")
		assert.NoError(t, err)
		defer os.Remove(f.Name())
		_, err = f.Write([]byte(tc.confStr))
		assert.NoError(t, err)

		v := viper.New()
		v.SetConfigFile(f.Name())
		err = v.ReadInConfig()
		if tc.shouldParse {
			assert.NoError(t, err)
		}
		success, resultAny, err := checkConfig(v)
		assert.NoError(t, err)
		result, ok := resultAny.(ConfigTestResult)
		assert.True(t, ok)
		if tc.isValid {
			assert.Empty(t, result.Error)
		} else {
			assert.NotEmpty(t, result.Error)
		}
		assert.Equal(t, tc.shouldParse, result.ParseSucceeded)
		assert.Equal(t, tc.isValid, result.IsValid)
		assert.Equal(t, tc.isValid && tc.shouldParse, success)
		assert.Equal(t, f.Name(), result.ConfigFile)
	}
}
