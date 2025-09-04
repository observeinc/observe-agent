package heartbeatreceiver

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver/internal/metadata"
	"go.opentelemetry.io/collector/consumer/consumertest"
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