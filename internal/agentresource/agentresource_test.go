package agentresource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/observeinc/observe-agent/internal/utils"
	"github.com/spf13/viper"
)

func TestAgentResource(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_agent_data.json")

	// Set up viper with custom path for this test
	originalPath := viper.GetString("agent_local_file_path")
	viper.Set("agent_local_file_path", testFilePath)
	defer viper.Set("agent_local_file_path", originalPath)

	// Test 1: Initialize with new file (file doesn't exist)
	agent1 := New()
	err := agent1.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize agent resource: %v", err)
	}

	// Verify agent instance ID was generated
	if agent1.GetAgentInstanceId() == "" {
		t.Error("Agent instance ID should not be empty")
	}

	// Verify agent start time was set
	if agent1.GetAgentStartTime() == 0 {
		t.Error("Agent start time should not be zero")
	}

	// Store the first agent's ID and start time
	firstAgentId := agent1.GetAgentInstanceId()
	firstStartTime := agent1.GetAgentStartTime()

	// Test 2: Initialize with existing file (should load same ID)
	agent2 := New()
	err = agent2.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize agent resource from existing file: %v", err)
	}

	// Verify same agent instance ID was loaded
	if agent2.GetAgentInstanceId() != firstAgentId {
		t.Errorf("Expected agent ID %s, got %s", firstAgentId, agent2.GetAgentInstanceId())
	}

	// Verify start time was set (may be same if run within same second)
	if agent2.GetAgentStartTime() < firstStartTime {
		t.Error("Agent start time should not go backwards")
	}
}

func TestDefaultPath(t *testing.T) {
	path := utils.GetDefaultAgentPath()
	if path == "" {
		t.Error("Default agent path should not be empty")
	}

	// Verify it returns expected paths for different OSes
	switch os := os.Getenv("GOOS"); os {
	case "darwin":
		if path != "/usr/local/observe-agent" {
			t.Errorf("Unexpected path for darwin: %s", path)
		}
	case "linux":
		if path != "/etc/observe-agent" {
			t.Errorf("Unexpected path for linux: %s", path)
		}
	}
}

func TestAgentResourceWithConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom_agent_data.json")

	// Store original value and restore after test
	originalPath := viper.GetString("agent_local_file_path")
	defer viper.Set("agent_local_file_path", originalPath)
	
	// Set custom path
	viper.Set("agent_local_file_path", configPath)

	// Test that New() uses the configured path
	agent := New()
	if agent.filePath != configPath {
		t.Errorf("Expected file path %s, got %s", configPath, agent.filePath)
	}

	// Test initialization
	err := agent.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize agent resource: %v", err)
	}

	// Verify file was created at configured location
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Agent local data file was not created at configured path")
	}

	// Verify agent instance ID was generated
	if agent.GetAgentInstanceId() == "" {
		t.Error("Agent instance ID should not be empty")
	}
}