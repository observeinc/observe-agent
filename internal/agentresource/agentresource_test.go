package agentresource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestAgentResource(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFilePath := filepath.Join(tempDir, "test_agent_data.json")

	// Set up viper with custom path for this test
	originalPath := viper.GetString("agent_local_file_path")
	viper.Set("agent_local_file_path", testFilePath)
	t.Cleanup(func() {
		viper.Set("agent_local_file_path", originalPath)
	})

	// Test 1: Initialize with new file (file doesn't exist)
	agent1, err := New()
	assert.NoError(t, err, "Failed to create agent resource")

	// Verify agent instance ID was generated
	assert.NotEmpty(t, agent1.GetAgentInstanceId(), "Agent instance ID should not be empty")

	// Verify agent start time was set
	assert.NotZero(t, agent1.GetAgentStartTime(), "Agent start time should not be zero")

	// Store the first agent's ID and start time
	firstAgentId := agent1.GetAgentInstanceId()
	firstStartTime := agent1.GetAgentStartTime()

	// Test 2: Initialize with existing file (should load same ID)
	agent2, err := New()
	assert.NoError(t, err, "Failed to create agent resource from existing file")

	// Verify same agent instance ID was loaded
	assert.Equal(t, firstAgentId, agent2.GetAgentInstanceId(), "Agent instance ID should be the same")

	// Verify start time was set (may be same if run within same second)
	assert.LessOrEqual(t, firstStartTime, agent2.GetAgentStartTime(), "Agent start time should not go backwards")
}

func TestAgentResourceWithConfig(t *testing.T) {
	// Create a temporary directory for testing
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "custom_agent_data.json")

	// Store original value and restore after test
	originalPath := viper.GetString("agent_local_file_path")
	t.Cleanup(func() {
		viper.Set("agent_local_file_path", originalPath)
	})

	// Set custom path
	viper.Set("agent_local_file_path", configPath)

	// Test that New() uses the configured path
	agent, err := New()
	assert.NoError(t, err, "Failed to create agent resource")

	// Verify file path is set correctly
	assert.Equal(t, configPath, agent.filePath, "Agent file path should match configured path")

	// Verify file was created at configured location
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Agent local data file was not created at configured path")
	}

	// Verify agent instance ID was generated
	assert.NotEmpty(t, agent.GetAgentInstanceId(), "Agent instance ID should not be empty")
}
