package agentresource

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/observeinc/observe-agent/internal/utils"
	"github.com/spf13/viper"
)

type AgentLocalData struct {
	AgentInstanceId string `json:"agent_instance_id"`
	AgentStartTime  int64  `json:"agent_start_time"`
}

type AgentResource struct {
	data     *AgentLocalData
	filePath string
}

const nameSuffixCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var defaultLocalFilePath = filepath.Join(utils.GetDefaultAgentPath(), "agent_local_data.json")

func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = nameSuffixCharset[rand.Intn(len(nameSuffixCharset))]
	}
	return string(b)
}

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
	a.data.AgentStartTime = time.Now().Unix()

	err := a.parseLocalFile()

	if err != nil && os.IsNotExist(err) {
		a.data.AgentInstanceId = a.generateAgentInstanceId()

		if err := a.persistToLocalFile(); err != nil {
			return fmt.Errorf("failed to persist agent data: %w", err)
		}
		return nil
	}

	if err != nil {
		return fmt.Errorf("failed to parse local file: %w", err)
	}

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

func (a *AgentResource) generateAgentInstanceId() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("agent-%s-%s", hostname, generateRandomString(6))
}

func (a *AgentResource) persistToLocalFile() error {
	jsonData, err := json.Marshal(a.data)
	if err != nil {
		return err
	}

	dir := filepath.Dir(a.filePath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	return os.WriteFile(a.filePath, jsonData, 0600)
}

func (a *AgentResource) parseLocalFile() error {
	if _, err := os.Stat(a.filePath); err != nil {
		return err
	}

	jsonData, err := os.ReadFile(a.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, a.data)
}
