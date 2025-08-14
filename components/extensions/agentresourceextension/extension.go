package agentresourceextension

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.uber.org/zap"
)

type AgentLocalData struct {
	AgentInstanceId string `json:"agent_instance_id"`
	AgentStartTime  int64  `json:"agent_start_time"`
}

type agentResourceExtension struct {
	cfg      *Config
	logger   *zap.Logger
	data     *AgentLocalData
	filePath string
}

var _ AgentResourceProvider = (*agentResourceExtension)(nil)

const nameSuffixCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

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

func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = nameSuffixCharset[rand.Intn(len(nameSuffixCharset))]
	}
	return string(b)
}

func newAgentResourceExtension(cfg *Config, logger *zap.Logger) *agentResourceExtension {
	filePath := cfg.LocalFilePath
	if filePath == "" {
		filePath = GetDefaultAgentPath() + "/agent_local_data.json"
	}

	return &agentResourceExtension{
		cfg:      cfg,
		logger:   logger,
		data:     &AgentLocalData{},
		filePath: filePath,
	}
}

func (e *agentResourceExtension) Start(ctx context.Context, host component.Host) error {
	e.logger.Info("Starting agent resource extension")
	return e.initializeAgentLocalData(ctx)
}

func (e *agentResourceExtension) Shutdown(ctx context.Context) error {
	e.logger.Info("Shutting down agent resource extension")
	return nil
}

func (e *agentResourceExtension) GetAgentInstanceId() string {
	return e.data.AgentInstanceId
}

func (e *agentResourceExtension) GetAgentStartTime() int64 {
	return e.data.AgentStartTime
}

func (e *agentResourceExtension) GetAgentData() AgentLocalData {
	return *e.data
}

func (e *agentResourceExtension) generateAgentInstanceId() string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("agent-%s-%s", hostname, generateRandomString(6))
}

func (e *agentResourceExtension) persistToLocalFile() error {
	jsonData, err := json.Marshal(e.data)
	if err != nil {
		return err
	}

	dir := filepath.Dir(e.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(e.filePath, jsonData, 0644)
}

func (e *agentResourceExtension) parseLocalFile() error {
	if _, err := os.Stat(e.filePath); err != nil {
		return err
	}

	jsonData, err := os.ReadFile(e.filePath)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonData, e.data)
}

func (e *agentResourceExtension) initializeAgentLocalData(ctx context.Context) error {
	e.data.AgentStartTime = time.Now().Unix()

	err := e.parseLocalFile()

	if err != nil && os.IsNotExist(err) {
		e.data.AgentInstanceId = e.generateAgentInstanceId()

		if err := e.persistToLocalFile(); err != nil {
			return err
		}
		e.logger.Info("Generated new agent instance ID", zap.String("agent_instance_id", e.data.AgentInstanceId))
		return nil
	}

	if err != nil {
		return err
	}

	e.logger.Info("Loaded existing agent instance ID", zap.String("agent_instance_id", e.data.AgentInstanceId))
	return nil
}