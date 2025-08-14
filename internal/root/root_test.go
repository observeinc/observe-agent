package root

import (
	"os"
	"testing"
)

func TestSetEnvVars(t *testing.T) {
	// Save original env var if it exists
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	defer func() {
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		}
	}()

	// Call setEnvVars which should initialize agent resource and set env var
	err := setEnvVars()
	if err != nil {
		t.Fatalf("setEnvVars failed: %v", err)
	}

	// Check that OBSERVE_AGENT_INSTANCE_ID was set
	agentID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	if agentID == "" {
		t.Error("OBSERVE_AGENT_INSTANCE_ID environment variable was not set")
	}

	// Verify the format of the agent ID (should be "agent-<hostname>-<random>")
	if len(agentID) < 10 || agentID[:6] != "agent-" {
		t.Errorf("Invalid agent ID format: %s", agentID)
	}
}