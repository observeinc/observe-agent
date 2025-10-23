package heartbeatreceiver

import (
	"encoding/base64"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactAndEncodeConfig(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		checkResult func(t *testing.T, result string)
	}{
		{
			name:  "obfuscates unquoted token",
			input: "token: abc123def456ghi789\n",
			checkResult: func(t *testing.T, result string) {
				decoded, err := base64.StdEncoding.DecodeString(result)
				require.NoError(t, err)
				decodedStr := string(decoded)
				// Should show first 8 chars
				assert.Contains(t, decodedStr, "abc123de")
				// Should have asterisks
				assert.Contains(t, decodedStr, "***")
				// Full token should NOT be present
				assert.NotContains(t, decodedStr, "abc123def456ghi789")
			},
		},
		{
			name:  "obfuscates double-quoted token",
			input: "token: \"abc123def456ghi789\"\n",
			checkResult: func(t *testing.T, result string) {
				decoded, err := base64.StdEncoding.DecodeString(result)
				require.NoError(t, err)
				decodedStr := string(decoded)
				// Should show first 8 chars
				assert.Contains(t, decodedStr, "abc123de")
				// Should have asterisks
				assert.Contains(t, decodedStr, "***")
				// Full token should NOT be present
				assert.NotContains(t, decodedStr, "abc123def456ghi789")
			},
		},
		{
			name: "handles multi-line config with token",
			input: `observe_url: https://example.com
token: abc123def456ghi789
debug: true
`,
			checkResult: func(t *testing.T, result string) {
				decoded, err := base64.StdEncoding.DecodeString(result)
				require.NoError(t, err)
				decodedStr := string(decoded)
				// Non-sensitive fields should be unchanged
				assert.Contains(t, decodedStr, "observe_url: https://example.com")
				assert.Contains(t, decodedStr, "debug: true")
				// Token should show first 8 chars
				assert.Contains(t, decodedStr, "abc123de")
				// Should have asterisks
				assert.Contains(t, decodedStr, "***")
				// Full token should NOT be present
				assert.NotContains(t, decodedStr, "abc123def456ghi789")
			},
		},
		{
			name:        "returns error for empty content",
			input:       "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := redactAndEncodeConfig(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.NotEmpty(t, result)

			if tt.checkResult != nil {
				tt.checkResult(t, result)
			}
		})
	}
}

func TestRedactAndEncodeConfigWithCustomPatterns(t *testing.T) {
	// Tests extensibility: custom paths, prefix lengths, and multiple patterns
	tests := []struct {
		name        string
		patterns    []SensitiveFieldPattern
		input       string
		checkResult func(t *testing.T, decoded string)
	}{
		{
			name: "nested field with custom prefix length",
			patterns: []SensitiveFieldPattern{
				{Path: "database.password", PrefixLength: 4},
			},
			input: `database:
  host: localhost
  password: secretpassword123
  port: 5432
`,
			checkResult: func(t *testing.T, decoded string) {
				assert.Contains(t, decoded, "host: localhost")
				assert.Contains(t, decoded, "secr")
				assert.Contains(t, decoded, "***")
				assert.Contains(t, decoded, "port: 5432")
			},
		},
		{
			name: "multiple fields with different prefix lengths",
			patterns: []SensitiveFieldPattern{
				{Path: "token", PrefixLength: 8},
				{Path: "api_key", PrefixLength: 6},
			},
			input: `token: abc123def456ghi789
api_key: myapikey12345
observe_url: https://example.com
`,
			checkResult: func(t *testing.T, decoded string) {
				assert.Contains(t, decoded, "abc123de")
				assert.Contains(t, decoded, "myapik")
				assert.Contains(t, decoded, "observe_url: https://example.com")
			},
		},
		{
			name: "regex pattern matches authorization at any level (lowercase and capitalized)",
			patterns: []SensitiveFieldPattern{
				{KeyPattern: "^[aA]uthorization$", PrefixLength: 16},
			},
			input: `exporters:
  otlp:
    endpoint: https://example.com
    headers:
      authorization: Bearer token123456789abcdefghij
      content-type: application/json
receivers:
  http:
    headers:
      Authorization: Basic user:pass123456789xyz
`,
			checkResult: func(t *testing.T, decoded string) {
				// Non-sensitive fields should be unchanged
				assert.Contains(t, decoded, "endpoint: https://example.com")
				assert.Contains(t, decoded, "content-type: application/json")

				// Both authorization fields should be redacted with 16 char prefix
				// lowercase "authorization"
				assert.Contains(t, decoded, "Bearer token123")
				assert.Contains(t, decoded, "***")

				// capitalized "Authorization"
				assert.Contains(t, decoded, "Basic user:pass1")

				// Make sure the full values are NOT present
				assert.NotContains(t, decoded, "token123456789abcdefghij")
				assert.NotContains(t, decoded, "pass123456789xyz")
			},
		},
		{
			name: "regex pattern matches multiple occurrences at different depths",
			patterns: []SensitiveFieldPattern{
				{KeyPattern: "^password$", PrefixLength: 4},
			},
			input: `database:
  host: localhost
  password: dbpass123456
services:
  redis:
    password: redispass789
  postgres:
    password: pgpass456789
`,
			checkResult: func(t *testing.T, decoded string) {
				assert.Contains(t, decoded, "host: localhost")
				// All three passwords should be redacted with 4 char prefix
				assert.Contains(t, decoded, "dbpa")
				assert.Contains(t, decoded, "redi")
				assert.Contains(t, decoded, "pgpa")
				assert.Contains(t, decoded, "***")
				// Full passwords should not be present
				assert.NotContains(t, decoded, "dbpass123456")
				assert.NotContains(t, decoded, "redispass789")
				assert.NotContains(t, decoded, "pgpass456789")
			},
		},
		{
			name: "regex pattern with wildcard matches any key containing pattern",
			patterns: []SensitiveFieldPattern{
				{KeyPattern: ".*secret.*", PrefixLength: 6},
			},
			input: `config:
  api_secret: mysecret123456
  db_secret_key: dbsecret789
  public_key: publicvalue123
`,
			checkResult: func(t *testing.T, decoded string) {
				// Fields containing "secret" should be redacted
				assert.Contains(t, decoded, "mysecr")
				assert.Contains(t, decoded, "dbsecr")
				assert.Contains(t, decoded, "***")
				// public_key should NOT be redacted
				assert.Contains(t, decoded, "public_key: publicvalue123")
				// Full secret values should not be present
				assert.NotContains(t, decoded, "mysecret123456")
				assert.NotContains(t, decoded, "dbsecret789")
			},
		},
		{
			name: "case-insensitive regex pattern",
			patterns: []SensitiveFieldPattern{
				{KeyPattern: "(?i)^authorization$", PrefixLength: 8},
			},
			input: `headers:
  Authorization: Bearer token123
  AUTHORIZATION: Basic user:pass
  authorization: Token abc123
`,
			checkResult: func(t *testing.T, decoded string) {
				// All variations should be redacted
				assert.Contains(t, decoded, "Bearer t")
				assert.Contains(t, decoded, "Basic us")
				assert.Contains(t, decoded, "Token ab")
				assert.Contains(t, decoded, "***")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily replace the global patterns
			originalPatterns := sensitiveFieldPatterns
			defer func() {
				sensitiveFieldPatterns = originalPatterns
			}()

			// Initialize the patterns (compile regex)
			sensitiveFieldPatterns = tt.patterns
			for i := range sensitiveFieldPatterns {
				if sensitiveFieldPatterns[i].KeyPattern != "" {
					// Compile the regex pattern
					re, err := regexp.Compile(sensitiveFieldPatterns[i].KeyPattern)
					if err != nil {
						// If the pattern is invalid, treat it as a literal string match
						sensitiveFieldPatterns[i].keyRegex = regexp.MustCompile("^" + regexp.QuoteMeta(sensitiveFieldPatterns[i].KeyPattern) + "$")
					} else {
						sensitiveFieldPatterns[i].keyRegex = re
					}
				}
			}

			result, err := redactAndEncodeConfig(tt.input)
			require.NoError(t, err)

			decoded, err := base64.StdEncoding.DecodeString(result)
			require.NoError(t, err)

			if tt.checkResult != nil {
				tt.checkResult(t, string(decoded))
			}
		})
	}
}

func TestRedactAndEncodeConfigWithDefaultPatterns(t *testing.T) {
	// Test with the actual default patterns used in production
	input := `observe_url: https://collect.observeinc.com
token: ds_abc123:VeryLongSecretToken1234567890
self_monitoring:
  enabled: true
exporters:
  otlp:
    endpoint: https://example.com
    headers:
      authorization: Bearer myBearerToken123456789
      Authorization: Basic myBasicAuth987654321
      content-type: application/json
`

	result, err := redactAndEncodeConfig(input)
	require.NoError(t, err)

	decoded, err := base64.StdEncoding.DecodeString(result)
	require.NoError(t, err)
	decodedStr := string(decoded)

	// Non-sensitive fields should be unchanged
	assert.Contains(t, decodedStr, "observe_url: https://collect.observeinc.com")
	assert.Contains(t, decodedStr, "enabled: true")
	assert.Contains(t, decodedStr, "endpoint: https://example.com")
	assert.Contains(t, decodedStr, "content-type: application/json")

	// Token should be redacted with 8 char prefix
	assert.Contains(t, decodedStr, "ds_abc12")
	assert.Contains(t, decodedStr, "***")
	assert.NotContains(t, decodedStr, "VeryLongSecretToken1234567890")

	// Both authorization headers should be redacted with 16 char prefix
	assert.Contains(t, decodedStr, "Bearer myBearerT")
	assert.Contains(t, decodedStr, "Basic myBasicAut")

	// Full auth values should NOT be present
	assert.NotContains(t, decodedStr, "myBearerToken123456789")
	assert.NotContains(t, decodedStr, "myBasicAuth987654321")
}

func TestObfuscateValue(t *testing.T) {
	tests := []struct {
		name         string
		value        string
		prefixLength int
		expected     string
	}{
		{
			name:         "obfuscates with standard prefix",
			value:        "secret123456789",
			prefixLength: 8,
			expected:     "secret12*******",
		},
		{
			name:         "obfuscates short value",
			value:        "short",
			prefixLength: 8,
			expected:     "*****",
		},
		{
			name:         "obfuscates with zero prefix",
			value:        "topsecret",
			prefixLength: 0,
			expected:     "*********",
		},
		{
			name:         "handles empty value",
			value:        "",
			prefixLength: 8,
			expected:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := obfuscateValue(tt.value, tt.prefixLength)
			assert.Equal(t, tt.expected, result)
		})
	}
}
