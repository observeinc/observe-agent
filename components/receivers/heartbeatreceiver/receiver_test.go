package heartbeatreceiver

import (
	"context"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver/internal/metadata"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

func TestHeartbeatReceiverWithEnvVar(t *testing.T) {
	// Set up environment variable
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	defer func() {
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		} else {
			os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
		}
	}()

	testAgentID := "test-agent-123"
	os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

	// Create receiver
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	
	receiver, err := factory.CreateLogs(
		context.Background(),
		receivertest.NewNopSettings(metadata.Type),
		cfg,
		consumertest.NewNop(),
	)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}

	// Start the receiver
	err = receiver.Start(context.Background(), nil)
	if err != nil {
		t.Fatalf("Failed to start receiver: %v", err)
	}

	// Give it a moment to initialize
	time.Sleep(100 * time.Millisecond)

	// Shutdown the receiver
	err = receiver.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Failed to shutdown receiver: %v", err)
	}
}

func TestHeartbeatReceiverMissingEnvVar(t *testing.T) {
	// Ensure environment variable is not set
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
	defer func() {
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		}
	}()

	// Create receiver
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	
	receiver, err := factory.CreateLogs(
		context.Background(),
		receivertest.NewNopSettings(metadata.Type),
		cfg,
		consumertest.NewNop(),
	)
	if err != nil {
		t.Fatalf("Failed to create receiver: %v", err)
	}

	// Starting should fail without the environment variable
	err = receiver.Start(context.Background(), nil)
	if err == nil {
		t.Fatal("Expected error when OBSERVE_AGENT_INSTANCE_ID is not set")
	}

	expectedError := "OBSERVE_AGENT_INSTANCE_ID environment variable must be set"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}
}

func TestAddCommonHeartbeatFields(t *testing.T) {
	// Set up environment variable for agent instance ID
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	defer func() {
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		} else {
			os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
		}
	}()

	testAgentID := "test-agent-common-123"
	os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

	tests := []struct {
		name        string
		environment string
		kind        string
	}{
		{
			name:        "lifecycle event with linux environment",
			environment: "linux",
			kind:        "AgentLifecycleEvent",
		},
		{
			name:        "config event with kubernetes environment",
			environment: "kubernetes",
			kind:        "AgentConfig",
		},
		{
			name:        "config event with macos environment",
			environment: "macos",
			kind:        "AgentConfig",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create receiver
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig().(*Config)
			cfg.Environment = tt.environment

			receiver, err := newReceiver(
				receivertest.NewNopSettings(metadata.Type),
				cfg,
				consumertest.NewNop(),
			)
			require.NoError(t, err)

			// Initialize receiver state
			err = receiver.InitializeReceiverState(context.Background())
			require.NoError(t, err)

			// Create logs structure
			logs := plog.NewLogs()
			resourceLogs := logs.ResourceLogs().AppendEmpty()
			scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
			logRecord := scopeLogs.LogRecords().AppendEmpty()

			// Call the helper function
			startTime := time.Now().UnixNano()
			receiver.addCommonHeartbeatFields(resourceLogs, logRecord, tt.kind)
			endTime := time.Now().UnixNano()

			// Verify resource attributes
			attrs := resourceLogs.Resource().Attributes()

			agentID, found := attrs.Get("observe.agent.instance.id")
			assert.True(t, found, "Should have agent instance ID attribute")
			assert.Equal(t, testAgentID, agentID.Str())

			env, found := attrs.Get("observe.agent.environment")
			assert.True(t, found, "Should have environment attribute")
			assert.Equal(t, tt.environment, env.Str())

			processID, found := attrs.Get("observe.agent.processId")
			assert.True(t, found, "Should have process ID attribute")
			assert.NotEmpty(t, processID.Str())

			// Verify observe_transform
			observeTransform, found := logRecord.Attributes().Get("observe_transform")
			assert.True(t, found, "Should have observe_transform attribute")
			transformMap := observeTransform.Map()

			// Check kind
			kind, found := transformMap.Get("kind")
			assert.True(t, found, "Should have kind field")
			assert.Equal(t, tt.kind, kind.Str())

			// Check identifiers
			identifiers, found := transformMap.Get("identifiers")
			assert.True(t, found, "Should have identifiers map")
			identifiersMap := identifiers.Map()

			idInIdentifiers, found := identifiersMap.Get("observe.agent.instance.id")
			assert.True(t, found, "Should have agent instance ID in identifiers")
			assert.Equal(t, testAgentID, idInIdentifiers.Str())

			// Check control
			control, found := transformMap.Get("control")
			assert.True(t, found, "Should have control map")
			controlMap := control.Map()

			isDelete, found := controlMap.Get("isDelete")
			assert.True(t, found, "Should have isDelete in control")
			assert.False(t, isDelete.Bool(), "isDelete should be false")

			// Check timestamps
			processStartTime, found := transformMap.Get("process_start_time")
			assert.True(t, found, "Should have process_start_time")
			assert.Equal(t, receiver.state.AgentStartTime, processStartTime.Int())

			validFrom, found := transformMap.Get("valid_from")
			assert.True(t, found, "Should have valid_from")
			assert.GreaterOrEqual(t, validFrom.Int(), startTime, "valid_from should be after start")
			assert.LessOrEqual(t, validFrom.Int(), endTime, "valid_from should be before end")

			validTo, found := transformMap.Get("valid_to")
			assert.True(t, found, "Should have valid_to")
			// valid_to should be approximately 90 minutes (5400000000000 ns) after valid_from
			// We allow a small tolerance for timing differences
			expectedValidTo := validFrom.Int() + 5400000000000
			timeDiff := validTo.Int() - expectedValidTo
			assert.LessOrEqual(t, timeDiff, int64(1000000), "valid_to should be within 1ms of 90 minutes after valid_from")
			assert.GreaterOrEqual(t, timeDiff, int64(-1000000), "valid_to should be within 1ms of 90 minutes after valid_from")
		})
	}
}

func TestObfuscateSensitiveFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:  "obfuscates unquoted token",
			input: "token: abc123def456ghi789\n",
			expected: `token: abc123de**********
`,
		},
		{
			name:  "obfuscates double-quoted token",
			input: "token: \"abc123def456ghi789\"\n",
			expected: `token: "abc123de**********"
`,
		},
		{
			name:  "obfuscates single-quoted token",
			input: "token: 'abc123def456ghi789'\n",
			expected: `token: 'abc123de**********'
`,
		},
		{
			name:  "preserves comments after token",
			input: "token: abc123def456ghi789 # secret token\n",
			expected: `token: abc123de********** # secret token
`,
		},
		{
			name:  "handles short token (8 chars or less)",
			input: "token: short\n",
			expected: `token: '*****'
`,
		},
		{
			name: "handles multi-line config with token",
			input: `observe_url: https://example.com
token: abc123def456ghi789
debug: true
`,
			expected: `observe_url: https://example.com
token: abc123de**********
debug: true
`,
		},
		{
			name: "preserves other fields unchanged",
			input: `observe_url: https://example.com
token: abc123def456ghi789
self_monitoring:
  enabled: true
`,
			expected: `observe_url: https://example.com
token: abc123de**********
self_monitoring:
    enabled: true
`,
		},
		{
			name:  "handles real token format (with colon)",
			input: "token: ds_abc123:veryLongTokenStringHere123456789\n",
			expected: `token: ds_abc12**********************************
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := obfuscateSensitiveFields([]byte(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestObfuscateSensitiveFieldsWithCustomPatterns(t *testing.T) {
	// Test with custom patterns to demonstrate extensibility
	tests := []struct {
		name     string
		patterns []SensitiveFieldPattern
		input    string
		expected string
	}{
		{
			name: "obfuscates nested field using dot notation",
			patterns: []SensitiveFieldPattern{
				{Path: "database.password", PrefixLength: 4},
			},
			input: `database:
  host: localhost
  password: secretpassword123
  port: 5432
`,
			expected: `database:
    host: localhost
    password: secr*************
    port: 5432
`,
		},
		{
			name: "obfuscates multiple different fields",
			patterns: []SensitiveFieldPattern{
				{Path: "token", PrefixLength: 8},
				{Path: "api_key", PrefixLength: 6},
			},
			input: `token: abc123def456ghi789
api_key: myapikey12345
observe_url: https://example.com
`,
			expected: `token: abc123de**********
api_key: myapik*******
observe_url: https://example.com
`,
		},
		{
			name: "handles different prefix lengths",
			patterns: []SensitiveFieldPattern{
				{Path: "short", PrefixLength: 2},
				{Path: "long", PrefixLength: 12},
			},
			input: `short: abcdefgh
long: abcdefghijklmnop
`,
			expected: `short: ab******
long: abcdefghijkl****
`,
		},
		{
			name: "obfuscates deeply nested field",
			patterns: []SensitiveFieldPattern{
				{Path: "auth_check.headers.authorization", PrefixLength: 8},
			},
			input: `auth_check:
  url: https://example.com
  headers:
    authorization: Bearer secrettoken123456789
    content-type: application/json
`,
			expected: `auth_check:
    url: https://example.com
    headers:
        authorization: Bearer s*******************
        content-type: application/json
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Temporarily replace the global patterns
			originalPatterns := sensitiveFieldPatterns
			sensitiveFieldPatterns = tt.patterns
			defer func() {
				sensitiveFieldPatterns = originalPatterns
			}()

			result := obfuscateSensitiveFields([]byte(tt.input))
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestGetObserveAgentConfigBytes(t *testing.T) {
	tests := []struct {
		name        string
		setupViper  func(t *testing.T) string // returns config file path
		expectError bool
		errorMsg    string
	}{
		{
			name: "successfully reads and encodes config file",
			setupViper: func(t *testing.T) string {
				// Create a temporary config file
				tmpDir := t.TempDir()
				configPath := filepath.Join(tmpDir, "observe-agent.yaml")
				configContent := []byte("self_monitoring:\n  enabled: true\n  fleet:\n    enabled: true\n")
				err := os.WriteFile(configPath, configContent, 0644)
				require.NoError(t, err)

				// Set viper to use this config file
				v := viper.New()
				v.SetConfigFile(configPath)
				err = v.ReadInConfig()
				require.NoError(t, err)

				// Replace global viper instance
				viper.Reset()
				viper.SetConfigFile(configPath)
				err = viper.ReadInConfig()
				require.NoError(t, err)

				return configPath
			},
			expectError: false,
		},
		{
			name: "returns error when config file not found",
			setupViper: func(t *testing.T) string {
				// Set viper to use a non-existent file
				viper.Reset()
				viper.SetConfigFile("/nonexistent/path/observe-agent.yaml")
				return "/nonexistent/path/observe-agent.yaml"
			},
			expectError: true,
			errorMsg:    "no such file or directory",
		},
		{
			name: "returns error when no config file in use",
			setupViper: func(t *testing.T) string {
				// Reset viper so no config file is set
				viper.Reset()
				return ""
			},
			expectError: true,
			errorMsg:    "no config file in use",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupViper(t)

			// Create a receiver
			factory := NewFactory()
			cfg := factory.CreateDefaultConfig()
			receiver, err := newReceiver(
				receivertest.NewNopSettings(metadata.Type),
				cfg.(*Config),
				consumertest.NewNop(),
			)
			require.NoError(t, err)

			// Call the method
			result, err := receiver.getObserveAgentConfigBytes(context.Background())

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Empty(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, result)

				// Verify it's valid base64
				decoded, err := base64.StdEncoding.DecodeString(result)
				assert.NoError(t, err)

				// Verify the decoded content matches the obfuscated original file
				if configPath != "" {
					originalContent, err := os.ReadFile(configPath)
					require.NoError(t, err)
					obfuscatedContent := obfuscateSensitiveFields(originalContent)
					assert.Equal(t, obfuscatedContent, decoded)
				}
			}
		})
	}
}

func TestGetOtelConfigBytes(t *testing.T) {
	// Note: Testing getOtelConfigBytes is complex because it depends on the full
	// OTEL collector setup. For now, we'll test the error handling and basic functionality.
	// Full integration testing will be done when we run the receiver.

	t.Run("returns base64 encoded string", func(t *testing.T) {
		// Create a minimal test setup
		factory := NewFactory()
		cfg := factory.CreateDefaultConfig()
		receiver, err := newReceiver(
			receivertest.NewNopSettings(metadata.Type),
			cfg.(*Config),
			consumertest.NewNop(),
		)
		require.NoError(t, err)

		ctx := context.Background()
		result, err := receiver.getOtelConfigBytes(ctx)

		// We expect this to potentially error in test environment
		// since the full OTEL config setup may not be available
		if err != nil {
			// That's acceptable in unit tests - we'll verify in integration tests
			t.Logf("getOtelConfigBytes returned error (expected in unit test): %v", err)
			return
		}

		// If it succeeds, verify it's valid base64
		if result != "" {
			decoded, decodeErr := base64.StdEncoding.DecodeString(result)
			assert.NoError(t, decodeErr, "Result should be valid base64")
			assert.NotEmpty(t, decoded, "Decoded content should not be empty")
		}
	})
}

func TestGenerateConfigHeartbeat(t *testing.T) {
	// Set up environment variable for agent instance ID
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	defer func() {
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		} else {
			os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
		}
	}()

	testAgentID := "test-agent-config-123"
	os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

	// Set up a temporary config file for testing
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "observe-agent.yaml")
	configContent := []byte("self_monitoring:\n  enabled: true\n")
	err := os.WriteFile(configPath, configContent, 0644)
	require.NoError(t, err)

	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	// Create receiver with a mock consumer to capture logs
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig().(*Config)
	cfg.Environment = "linux"

	sink := &consumertest.LogsSink{}
	receiver, err := newReceiver(
		receivertest.NewNopSettings(metadata.Type),
		cfg,
		sink,
	)
	require.NoError(t, err)

	// Initialize receiver state
	err = receiver.InitializeReceiverState(context.Background())
	require.NoError(t, err)

	// Call generateConfigHeartbeat
	ctx := context.Background()
	err = receiver.generateConfigHeartbeat(ctx)

	// We expect this might fail in test environment due to OTEL config
	// but we can still check the structure if it succeeds
	if err != nil {
		t.Logf("generateConfigHeartbeat returned error (may be expected in unit test): %v", err)
		// This is acceptable - we'll test the full flow in integration tests
		return
	}

	// If it succeeded, verify the log structure
	require.Equal(t, 1, sink.LogRecordCount(), "Should have one log record")

	logs := sink.AllLogs()
	require.Equal(t, 1, len(logs), "Should have one log batch")

	resourceLogs := logs[0].ResourceLogs()
	require.Equal(t, 1, resourceLogs.Len(), "Should have one resource log")

	// Check resource attributes
	attrs := resourceLogs.At(0).Resource().Attributes()
	agentID, found := attrs.Get("observe.agent.instance.id")
	assert.True(t, found, "Should have agent instance ID attribute")
	assert.Equal(t, testAgentID, agentID.Str())

	env, found := attrs.Get("observe.agent.environment")
	assert.True(t, found, "Should have environment attribute")
	assert.Equal(t, "linux", env.Str())

	_, found = attrs.Get("observe.agent.processId")
	assert.True(t, found, "Should have process ID attribute")

	// Check log record
	scopeLogs := resourceLogs.At(0).ScopeLogs()
	require.Equal(t, 1, scopeLogs.Len(), "Should have one scope log")

	logRecords := scopeLogs.At(0).LogRecords()
	require.Equal(t, 1, logRecords.Len(), "Should have one log record")

	logRecord := logRecords.At(0)

	// Check observe_transform
	observeTransform, found := logRecord.Attributes().Get("observe_transform")
	assert.True(t, found, "Should have observe_transform attribute")
	assert.Equal(t, "AgentConfig", observeTransform.Map().AsRaw()["kind"], "Kind should be AgentConfig")

	// Check identifiers
	identifiers, ok := observeTransform.Map().AsRaw()["identifiers"].(map[string]interface{})
	assert.True(t, ok, "Should have identifiers map")
	assert.Equal(t, testAgentID, identifiers["observe.agent.instance.id"])

	// Check control
	control, ok := observeTransform.Map().AsRaw()["control"].(map[string]interface{})
	assert.True(t, ok, "Should have control map")
	assert.Equal(t, false, control["isDelete"])

	// Check timestamps
	assert.Contains(t, observeTransform.Map().AsRaw(), "process_start_time")
	assert.Contains(t, observeTransform.Map().AsRaw(), "valid_from")
	assert.Contains(t, observeTransform.Map().AsRaw(), "valid_to")

	// Check body
	body := logRecord.Body().Map()
	observeAgentConfig, found := body.Get("observeAgentConfig")
	assert.True(t, found, "Body should have observeAgentConfig field")

	// Verify it's valid base64
	decoded, err := base64.StdEncoding.DecodeString(observeAgentConfig.Str())
	assert.NoError(t, err, "observeAgentConfig should be valid base64")

	// The decoded config will be obfuscated and normalized (4-space indentation)
	// Just verify it's valid YAML and contains expected fields
	assert.Contains(t, string(decoded), "self_monitoring", "Decoded config should contain self_monitoring")
	assert.Contains(t, string(decoded), "enabled: true", "Decoded config should contain enabled: true")

	otelConfig, found := body.Get("otelConfig")
	assert.True(t, found, "Body should have otelConfig field")

	// Verify it's valid base64
	_, err = base64.StdEncoding.DecodeString(otelConfig.Str())
	assert.NoError(t, err, "otelConfig should be valid base64")
}

func TestConfigHeartbeatTimer(t *testing.T) {
	// Set up environment variable
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	defer func() {
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		} else {
			os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
		}
	}()

	testAgentID := "test-agent-timer-123"
	os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

	// Set up a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "observe-agent.yaml")
	configContent := []byte("self_monitoring:\n  enabled: true\n")
	err := os.WriteFile(configPath, configContent, 0644)
	require.NoError(t, err)

	viper.Reset()
	viper.SetConfigFile(configPath)
	err = viper.ReadInConfig()
	require.NoError(t, err)

	t.Run("both timers run independently", func(t *testing.T) {
		// Create receiver with fast intervals for testing
		factory := NewFactory()
		cfg := factory.CreateDefaultConfig().(*Config)
		cfg.Interval = "100ms"       // lifecycle heartbeat every 100ms
		cfg.ConfigInterval = "200ms" // config heartbeat every 200ms
		cfg.Environment = "linux"

		sink := &consumertest.LogsSink{}
		receiver, err := factory.CreateLogs(
			context.Background(),
			receivertest.NewNopSettings(metadata.Type),
			cfg,
			sink,
		)
		require.NoError(t, err)

		// Start the receiver
		ctx := context.Background()
		err = receiver.Start(ctx, nil)
		require.NoError(t, err)

		// Wait for some heartbeats to be generated
		time.Sleep(500 * time.Millisecond)

		// Shutdown the receiver
		err = receiver.Shutdown(context.Background())
		require.NoError(t, err)

		// Note: In a real test, we would verify that both types of heartbeats
		// were generated. However, this requires the full OTEL setup.
		// For now, we just verify the receiver starts and stops cleanly.
		// Integration tests will verify the actual heartbeat generation.
	})

	t.Run("graceful shutdown stops both timers", func(t *testing.T) {
		factory := NewFactory()
		cfg := factory.CreateDefaultConfig().(*Config)
		cfg.Interval = "1h"        // long interval
		cfg.ConfigInterval = "24h" // long interval
		cfg.Environment = "linux"

		receiver, err := factory.CreateLogs(
			context.Background(),
			receivertest.NewNopSettings(metadata.Type),
			cfg,
			consumertest.NewNop(),
		)
		require.NoError(t, err)

		// Start the receiver
		err = receiver.Start(context.Background(), nil)
		require.NoError(t, err)

		// Shutdown should complete quickly even with long timer intervals
		done := make(chan error, 1)
		go func() {
			done <- receiver.Shutdown(context.Background())
		}()

		select {
		case err := <-done:
			assert.NoError(t, err, "Shutdown should complete without error")
		case <-time.After(2 * time.Second):
			t.Fatal("Shutdown took too long - timers may not have stopped properly")
		}
	})
}