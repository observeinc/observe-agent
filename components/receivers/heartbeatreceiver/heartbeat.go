package heartbeatreceiver

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"

	"go.opentelemetry.io/collector/component"
)

type HeartbeatReceiver struct {
	cfg *Config
}

type HeartbeatLocalData struct {
	AgentInstanceId string `json:"agent_instance_id"`
}

var localData HeartbeatLocalData

func GetDefaultAgentPath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return "/usr/local/observe-agent"
	case "windows":
		return os.ExpandEnv("$ProgramFiles\\Observe\\observe-agent")
	case "linux":
		return "/etc/observe-agent"
	default:
		return "/etc/observe-agent"
	}
}

var localDataFilePath = GetDefaultAgentPath() + "/heartbeat_local_data.json"

func (r *HeartbeatReceiver) Start(ctx context.Context, host component.Host) error {
	return nil
}

func (r *HeartbeatReceiver) Shutdown(ctx context.Context) error {
	return nil
}

func (r *HeartbeatReceiver) GenerateAgentInstanceId(ctx context.Context) string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return "agent-" + os.Hostname() + "-"
}

func (r *HeartbeatReceiver) PersistToLocalFile(ctx context.Context) error {
	// Marshal the localData to JSON
	jsonData, err := json.Marshal(localData)
	if err != nil {
		return err
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(localDataFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write the JSON data to the file
	return os.WriteFile(localDataFilePath, jsonData, 0644)
}

func (r *HeartbeatReceiver) ParseLocalFile(ctx context.Context) error {
	// Check if the file exists
	if _, err := os.Stat(localDataFilePath); os.IsNotExist(err) {
		// File doesn't exist, initialize with empty data
		localData = HeartbeatLocalData{}
		return nil
	}

	// Read the file
	jsonData, err := os.ReadFile(localDataFilePath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data into localData
	return json.Unmarshal(jsonData, &localData)
}
