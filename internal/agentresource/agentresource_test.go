package agentresource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/observeinc/observe-agent/internal/utils"
)

func TestAgentResource(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_agent_data.json")

	// Test 1: Initialize with new file (file doesn't exist)
	agent1 := New(testFilePath)
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
	agent2 := New(testFilePath)
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