package root

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
)

func TestSetEnvVars(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_agent_data.json")

	// Save original viper config and env var if they exist
	originalPath := viper.GetString("agent_local_file_path")
	originalID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")

	// Set up cleanup to restore original state
	t.Cleanup(func() {
		viper.Set("agent_local_file_path", originalPath)
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
		} else {
			os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
		}
	})

	// Set up viper with custom path for this test to avoid permission issues
	viper.Set("agent_local_file_path", testFilePath)

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
