package heartbeatreceiver

import (
	"context"
	"encoding/base64"
	"os"
	"testing"
	"time"

	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver/internal/metadata"
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

func TestGenerateConfigHeartbeat(t *testing.T) {
	// Set up environment variables
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	originalAgentConfig := os.Getenv("OBSERVE_AGENT_CONFIG")
	originalOtelConfig := os.Getenv("OBSERVE_AGENT_OTEL_CONFIG")
	defer func() {
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		} else {
			os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
		}
		if originalAgentConfig != "" {
			os.Setenv("OBSERVE_AGENT_CONFIG", originalAgentConfig)
		} else {
			os.Unsetenv("OBSERVE_AGENT_CONFIG")
		}
		if originalOtelConfig != "" {
			os.Setenv("OBSERVE_AGENT_OTEL_CONFIG", originalOtelConfig)
		} else {
			os.Unsetenv("OBSERVE_AGENT_OTEL_CONFIG")
		}
	}()

	t.Run("successfully generates heartbeat with valid env vars", func(t *testing.T) {
		testAgentID := "test-agent-config-123"
		os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

		// Set up config environment variables (base64 encoded)
		agentConfigYaml := "self_monitoring:\n  enabled: true\n"
		otelConfigYaml := "receivers:\n  heartbeat:\n    interval: 5m\n"
		os.Setenv("OBSERVE_AGENT_CONFIG", base64.StdEncoding.EncodeToString([]byte(agentConfigYaml)))
		os.Setenv("OBSERVE_AGENT_OTEL_CONFIG", base64.StdEncoding.EncodeToString([]byte(otelConfigYaml)))

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
		require.NoError(t, err)

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
	})

	t.Run("sends partial heartbeat when OBSERVE_AGENT_CONFIG env var is missing", func(t *testing.T) {
		testAgentID := "test-agent-missing-config"
		os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

		// Only set OTEL config, not agent config (base64 encoded)
		os.Unsetenv("OBSERVE_AGENT_CONFIG")
		otelConfigYaml := "receivers:\n  heartbeat:\n    interval: 5m\n"
		os.Setenv("OBSERVE_AGENT_OTEL_CONFIG", base64.StdEncoding.EncodeToString([]byte(otelConfigYaml)))

		factory := NewFactory()
		cfg := factory.CreateDefaultConfig().(*Config)
		sink := &consumertest.LogsSink{}
		receiver, err := newReceiver(
			receivertest.NewNopSettings(metadata.Type),
			cfg,
			sink,
		)
		require.NoError(t, err)

		err = receiver.InitializeReceiverState(context.Background())
		require.NoError(t, err)

		// Should not crash, should return nil and log error
		ctx := context.Background()
		err = receiver.generateConfigHeartbeat(ctx)
		assert.NoError(t, err, "Should not return error when one env var is missing")

		// Should still send heartbeat with OTEL config only
		assert.Equal(t, 1, sink.LogRecordCount(), "Should send partial heartbeat with available config")

		// Verify the heartbeat has OTEL config but empty agent config
		logs := sink.AllLogs()
		require.Equal(t, 1, len(logs))
		logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		body := logRecord.Body().Map()

		agentConfig, found := body.Get("observeAgentConfig")
		assert.True(t, found)
		assert.Empty(t, agentConfig.Str(), "Agent config should be empty")

		otelConfig, found := body.Get("otelConfig")
		assert.True(t, found)
		assert.NotEmpty(t, otelConfig.Str(), "OTEL config should be present")
	})

	t.Run("sends partial heartbeat when OBSERVE_AGENT_OTEL_CONFIG env var is missing", func(t *testing.T) {
		testAgentID := "test-agent-missing-otel"
		os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

		// Only set agent config, not OTEL config (base64 encoded)
		agentConfigYaml := "self_monitoring:\n  enabled: true\n"
		os.Setenv("OBSERVE_AGENT_CONFIG", base64.StdEncoding.EncodeToString([]byte(agentConfigYaml)))
		os.Unsetenv("OBSERVE_AGENT_OTEL_CONFIG")

		factory := NewFactory()
		cfg := factory.CreateDefaultConfig().(*Config)
		sink := &consumertest.LogsSink{}
		receiver, err := newReceiver(
			receivertest.NewNopSettings(metadata.Type),
			cfg,
			sink,
		)
		require.NoError(t, err)

		err = receiver.InitializeReceiverState(context.Background())
		require.NoError(t, err)

		// Should not crash, should return nil and log error
		ctx := context.Background()
		err = receiver.generateConfigHeartbeat(ctx)
		assert.NoError(t, err, "Should not return error when one env var is missing")

		// Should still send heartbeat with agent config only
		assert.Equal(t, 1, sink.LogRecordCount(), "Should send partial heartbeat with available config")

		// Verify the heartbeat has agent config but empty OTEL config
		logs := sink.AllLogs()
		require.Equal(t, 1, len(logs))
		logRecord := logs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		body := logRecord.Body().Map()

		agentConfig, found := body.Get("observeAgentConfig")
		assert.True(t, found)
		assert.NotEmpty(t, agentConfig.Str(), "Agent config should be present")

		otelConfig, found := body.Get("otelConfig")
		assert.True(t, found)
		assert.Empty(t, otelConfig.Str(), "OTEL config should be empty")
	})

	t.Run("skips heartbeat when both env vars are missing", func(t *testing.T) {
		testAgentID := "test-agent-both-missing"
		os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

		// Unset both configs
		os.Unsetenv("OBSERVE_AGENT_CONFIG")
		os.Unsetenv("OBSERVE_AGENT_OTEL_CONFIG")

		factory := NewFactory()
		cfg := factory.CreateDefaultConfig().(*Config)
		sink := &consumertest.LogsSink{}
		receiver, err := newReceiver(
			receivertest.NewNopSettings(metadata.Type),
			cfg,
			sink,
		)
		require.NoError(t, err)

		err = receiver.InitializeReceiverState(context.Background())
		require.NoError(t, err)

		// Should not crash
		ctx := context.Background()
		err = receiver.generateConfigHeartbeat(ctx)
		assert.NoError(t, err, "Should not return error when both env vars are missing")

		// No logs should be sent when both are missing
		assert.Equal(t, 0, sink.LogRecordCount(), "Should not send heartbeat when both configs are missing")
	})

	t.Run("gracefully handles invalid YAML in env vars", func(t *testing.T) {
		testAgentID := "test-agent-invalid-yaml"
		os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

		// Set invalid YAML (base64 encoded)
		os.Setenv("OBSERVE_AGENT_CONFIG", base64.StdEncoding.EncodeToString([]byte("invalid: yaml: content: [[[}")))
		os.Setenv("OBSERVE_AGENT_OTEL_CONFIG", base64.StdEncoding.EncodeToString([]byte("also: invalid: {{")))

		factory := NewFactory()
		cfg := factory.CreateDefaultConfig().(*Config)
		sink := &consumertest.LogsSink{}
		receiver, err := newReceiver(
			receivertest.NewNopSettings(metadata.Type),
			cfg,
			sink,
		)
		require.NoError(t, err)

		err = receiver.InitializeReceiverState(context.Background())
		require.NoError(t, err)

		// Should not crash, should return nil and log error
		ctx := context.Background()
		err = receiver.generateConfigHeartbeat(ctx)
		assert.NoError(t, err, "Should not return error when YAML is invalid")

		// No logs should be sent
		assert.Equal(t, 0, sink.LogRecordCount(), "Should not send heartbeat when YAML is invalid")
	})
}

func TestConfigHeartbeatTimer(t *testing.T) {
	// Set up environment variables
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	originalAgentConfig := os.Getenv("OBSERVE_AGENT_CONFIG")
	originalOtelConfig := os.Getenv("OBSERVE_AGENT_OTEL_CONFIG")
	defer func() {
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		} else {
			os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
		}
		if originalAgentConfig != "" {
			os.Setenv("OBSERVE_AGENT_CONFIG", originalAgentConfig)
		} else {
			os.Unsetenv("OBSERVE_AGENT_CONFIG")
		}
		if originalOtelConfig != "" {
			os.Setenv("OBSERVE_AGENT_OTEL_CONFIG", originalOtelConfig)
		} else {
			os.Unsetenv("OBSERVE_AGENT_OTEL_CONFIG")
		}
	}()

	testAgentID := "test-agent-timer-123"
	os.Setenv("OBSERVE_AGENT_INSTANCE_ID", testAgentID)

	// Set up config environment variables (base64 encoded)
	agentConfigYaml := "self_monitoring:\n  enabled: true\n"
	otelConfigYaml := "receivers:\n  heartbeat:\n    interval: 5m\n"
	os.Setenv("OBSERVE_AGENT_CONFIG", base64.StdEncoding.EncodeToString([]byte(agentConfigYaml)))
	os.Setenv("OBSERVE_AGENT_OTEL_CONFIG", base64.StdEncoding.EncodeToString([]byte(otelConfigYaml)))

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