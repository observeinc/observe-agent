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
	originalVersion := os.Getenv("OBSERVE_AGENT_VERSION")

	// Set up cleanup to restore original state
	t.Cleanup(func() {
		viper.Set("agent_local_file_path", originalPath)
		if originalID != "" {
			os.Setenv("OBSERVE_AGENT_INSTANCE_ID", originalID)
			os.Setenv("OBSERVE_AGENT_VERSION", originalVersion)
		} else {
			os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")
			os.Unsetenv("OBSERVE_AGENT_VERSION")
		}
	})

	// Set up viper with custom path for this test to avoid permission issues
	viper.Set("agent_local_file_path", testFilePath)
	// An inherited ID would take precedence over the default.
	os.Unsetenv("OBSERVE_AGENT_INSTANCE_ID")

	// Call setEnvVars which should initialize agent resource and set env var
	err := setEnvVars()
	if err != nil {
		t.Fatalf("setEnvVars failed: %v", err)
	}

	// Check that OBSERVE_AGENT_INSTANCE_ID was set
	agentID := os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	agentVersion := os.Getenv("OBSERVE_AGENT_VERSION")
	if agentID == "" {
		t.Error("OBSERVE_AGENT_INSTANCE_ID environment variable was not set")
	}
	if agentVersion == "" {
		t.Error("OBSERVE_AGENT_VERSION environment variable was not set")
	}

	// The agent ID defaults to the hostname when nothing else is configured.
	hostname, err := os.Hostname()
	if err != nil {
		t.Fatalf("failed to get hostname: %v", err)
	}
	if agentID != hostname {
		t.Errorf("Expected agent ID %q, got %q", hostname, agentID)
	}
}
