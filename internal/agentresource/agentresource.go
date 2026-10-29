package agentresource

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/observeinc/observe-agent/internal/utils"
	"github.com/spf13/viper"
)

// InstanceIdEnvVar is the environment variable the rendered collector config
// reads the agent instance ID from. Deployments that already know a stable
// identity for the agent process set it before the agent starts; the Helm chart
// uses it to pass the node name for daemonsets, the deployment name for
// singleton deployments, and the pod name otherwise.
const InstanceIdEnvVar = "OBSERVE_AGENT_INSTANCE_ID"

type AgentLocalData struct {
	AgentInstanceId string `json:"agent_instance_id"`
	AgentStartTime  int64  `json:"agent_start_time"`
}

type AgentResource struct {
	data     *AgentLocalData
	filePath string
}

var defaultLocalFilePath = filepath.Join(utils.GetDefaultAgentDataPath(), "agent_local_data.json")

func New() (*AgentResource, error) {
	var filePath string
	// Check if configured in viper
	configuredPath := viper.GetString("agent_local_file_path")
	if configuredPath != "" {
		filePath = configuredPath
	} else {
		filePath = defaultLocalFilePath
	}

	agentResource := &AgentResource{
		data:     &AgentLocalData{},
		filePath: filePath,
	}

	if err := agentResource.initialize(); err != nil {
		return nil, err
	}

	return agentResource, nil
}

func (a *AgentResource) initialize() error {
	a.data.AgentStartTime = time.Now().UnixNano()

	// A configured ID is authoritative and already stable across restarts, so it
	// never needs to be read back from or written to the local data file.
	if id := configuredInstanceId(); id != "" {
		a.data.AgentInstanceId = id
		return nil
	}

	// Agents installed before the ID became deterministic have a random suffix
	// that only exists in this file, so keep using it to preserve their identity.
	id, err := a.readInstanceIdFromLocalFile()
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		if errors.Is(err, fs.ErrPermission) {
			return fmt.Errorf("failed to parse local file: permission denied reading %s (check file permissions and ownership)", a.filePath)
		}
		return fmt.Errorf("failed to parse local file: %w", err)
	}
	if id == "" {
		id = defaultInstanceId()
	}

	a.data.AgentInstanceId = id
	return nil
}

func (a *AgentResource) GetAgentInstanceId() string {
	return a.data.AgentInstanceId
}

func (a *AgentResource) GetAgentStartTime() int64 {
	return a.data.AgentStartTime
}

func (a *AgentResource) GetAgentData() AgentLocalData {
	return *a.data
}

func (a *AgentResource) GetFilePath() string {
	return a.filePath
}

// configuredInstanceId returns the ID set explicitly by the user, either through
// the agent config or through the environment variable the collector config and
// the Helm chart use.
func configuredInstanceId() string {
	if id := viper.GetString("agent_instance_id"); id != "" {
		return id
	}
	return os.Getenv(InstanceIdEnvVar)
}

// defaultInstanceId identifies the agent by hostname, which is stable across
// restarts without having to be persisted. Environments where the hostname is
// neither stable nor unique per agent, such as Kubernetes pods, are expected to
// set InstanceIdEnvVar instead.
func defaultInstanceId() string {
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		return "unknown"
	}
	return hostname
}

func (a *AgentResource) readInstanceIdFromLocalFile() (string, error) {
	jsonData, err := os.ReadFile(a.filePath)
	if err != nil {
		return "", err
	}

	var data AgentLocalData
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return "", err
	}

	return data.AgentInstanceId, nil
}
