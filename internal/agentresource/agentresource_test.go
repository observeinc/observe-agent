package agentresource

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTest points the agent local data file at a temp path and clears any
// configured instance ID so each test starts from a known state.
func setupTest(t *testing.T) string {
	t.Helper()
	testFilePath := filepath.Join(t.TempDir(), "test_agent_data.json")

	originalPath := viper.GetString("agent_local_file_path")
	originalId := viper.GetString("agent_instance_id")
	originalEnvId := os.Getenv(InstanceIdEnvVar)
	t.Cleanup(func() {
		viper.Set("agent_local_file_path", originalPath)
		viper.Set("agent_instance_id", originalId)
		if originalEnvId != "" {
			os.Setenv(InstanceIdEnvVar, originalEnvId)
		} else {
			os.Unsetenv(InstanceIdEnvVar)
		}
	})

	viper.Set("agent_local_file_path", testFilePath)
	viper.Set("agent_instance_id", "")
	os.Unsetenv(InstanceIdEnvVar)

	return testFilePath
}

func TestAgentResourceDefaultsToHostname(t *testing.T) {
	setupTest(t)

	hostname, err := os.Hostname()
	require.NoError(t, err)

	agent, err := New()
	assert.NoError(t, err, "Failed to create agent resource")
	assert.Equal(t, hostname, agent.GetAgentInstanceId(), "Agent instance ID should default to the hostname")
	assert.NotZero(t, agent.GetAgentStartTime(), "Agent start time should not be zero")
}

func TestAgentResourceIsStableWithoutLocalFile(t *testing.T) {
	testFilePath := setupTest(t)

	agent1, err := New()
	assert.NoError(t, err)

	agent2, err := New()
	assert.NoError(t, err)

	// The hostname is stable across restarts, so the ID no longer has to be
	// written to disk to stay the same.
	assert.Equal(t, agent1.GetAgentInstanceId(), agent2.GetAgentInstanceId(), "Agent instance ID should be the same")
	assert.NoFileExists(t, testFilePath, "Agent local data file should not be written")
}

func TestAgentResourceReusesExistingLocalFileId(t *testing.T) {
	testFilePath := setupTest(t)

	// Agents installed before the ID became deterministic keep the ID that only
	// exists in their local data file.
	existingId := "agent-somehost-aB3xQ9"
	require.NoError(t, os.WriteFile(testFilePath, []byte(`{"agent_instance_id":"`+existingId+`","agent_start_time":1234567890}`), 0644))

	agent, err := New()
	assert.NoError(t, err)
	assert.Equal(t, existingId, agent.GetAgentInstanceId(), "Agent instance ID should come from the existing local file")
}

func TestAgentResourceConfiguredIdWins(t *testing.T) {
	testFilePath := setupTest(t)
	require.NoError(t, os.WriteFile(testFilePath, []byte(`{"agent_instance_id":"agent-somehost-aB3xQ9"}`), 0644))

	viper.Set("agent_instance_id", "configured-id")

	agent, err := New()
	assert.NoError(t, err)
	assert.Equal(t, "configured-id", agent.GetAgentInstanceId(), "Configured agent instance ID should take precedence")
}

func TestAgentResourceEnvIdWins(t *testing.T) {
	testFilePath := setupTest(t)
	require.NoError(t, os.WriteFile(testFilePath, []byte(`{"agent_instance_id":"agent-somehost-aB3xQ9"}`), 0644))

	// This is how the Helm chart passes the node, deployment, or pod name.
	os.Setenv(InstanceIdEnvVar, "some-node-name")

	agent, err := New()
	assert.NoError(t, err)
	assert.Equal(t, "some-node-name", agent.GetAgentInstanceId(), "Environment agent instance ID should take precedence")
}

func TestAgentResourceUnreadableLocalFile(t *testing.T) {
	testFilePath := setupTest(t)
	require.NoError(t, os.WriteFile(testFilePath, []byte("not json"), 0644))

	_, err := New()
	assert.ErrorContains(t, err, "failed to parse local file")
}

func TestAgentResourceWithConfig(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "custom_agent_data.json")

	originalPath := viper.GetString("agent_local_file_path")
	t.Cleanup(func() {
		viper.Set("agent_local_file_path", originalPath)
	})
	viper.Set("agent_local_file_path", configPath)

	agent, err := New()
	assert.NoError(t, err, "Failed to create agent resource")
	assert.Equal(t, configPath, agent.filePath, "Agent file path should match configured path")
	assert.NotEmpty(t, agent.GetAgentInstanceId(), "Agent instance ID should not be empty")
}
