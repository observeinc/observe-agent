package heartbeatreceiver

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
)

type HeartbeatReceiver struct {
	cfg          *Config
	settings     receiver.Settings
	obsrecv      *receiverhelper.ObsReport
	nextConsumer consumer.Logs
	ticker       *time.Ticker
	cancel       context.CancelFunc
}

type HeartbeatLocalData struct {
	AgentInstanceId string `json:"agent_instance_id"`
	AgentStartTime  int64
}

type HeartbeatLogRecord struct {
	AgentInstanceId string `json:"agent_instance_id"`
	AgentStartTime  int64  `json:"agent_start_time"`
}

var localData HeartbeatLocalData
var localDataFilePath = GetDefaultAgentPath() + "/heartbeat_local_data.json"

const nameSuffixCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

// generateRandomString generates a random alphanumeric string of the specified length
func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = nameSuffixCharset[rand.Intn(len(nameSuffixCharset))]
	}
	return string(b)
}

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

func newReceiver(set receiver.Settings, cfg *Config, consumer consumer.Logs) (*HeartbeatReceiver, error) {
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		LongLivedCtx:           true,
		ReceiverID:             set.ID,
		ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}

	return &HeartbeatReceiver{
		cfg:          cfg,
		settings:     set,
		obsrecv:      obsrecv,
		nextConsumer: consumer,
	}, nil
}

func (r *HeartbeatReceiver) Start(ctx context.Context, host component.Host) error {
	r.settings.Logger.Info("Starting heartbeat receiver")
	ctx, r.cancel = context.WithCancel(ctx)
	err := r.InitializeAgentLocalData(ctx)
	if err != nil {
		return err
	}
	interval, _ := time.ParseDuration(r.cfg.Interval)
	r.ticker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-r.ticker.C:
				// hb := HeartbeatLogRecord{
				// 	AgentInstanceId: localData.AgentInstanceId,
				// 	AgentStartTime:  localData.AgentStartTime,
				// }
				// jsonHb, _ := json.Marshal(hb)
				r.settings.Logger.Info("Sending heartbeat", zap.String("agent_instance_id", localData.AgentInstanceId))
				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().PutStr("agentInstanceId", localData.AgentInstanceId)

				scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
				logRecord := scopeLogs.LogRecords().AppendEmpty()
				body := logRecord.Body().SetEmptyMap()
				body.PutStr("agent_instance_id", localData.AgentInstanceId)
				body.PutInt("agent_start_time", localData.AgentStartTime)

				ctx := r.obsrecv.StartLogsOp(context.Background())
				err := r.nextConsumer.ConsumeLogs(context.Background(), logs)
				r.obsrecv.EndLogsOp(ctx, metadata.Type.String(), 1, err)
				if err != nil {
					r.settings.Logger.Error("failed to send logs: %w", zap.Error(err))
				}
			case <-ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (r *HeartbeatReceiver) Shutdown(ctx context.Context) error {
	r.settings.Logger.Info("Shutting down heartbeat receiver")
	if r.ticker != nil {
		r.ticker.Stop()
	}
	if r.cancel != nil {
		r.cancel()
	}

	return nil
}

func (r *HeartbeatReceiver) GenerateAgentInstanceId(ctx context.Context) string {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return fmt.Sprintf("agent-%s-%s", hostname, generateRandomString(6))
}

func (r *HeartbeatReceiver) PersistToLocalFile(ctx context.Context) error {
	// Marshal the localData to JSON
	jsonData, err := json.Marshal(localData)
	if err != nil {
		return err
	}

	// Create the directory if it doesn't exist
	dir := filepath.Dir(r.cfg.LocalFilePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Write the JSON data to the file
	return os.WriteFile(r.cfg.LocalFilePath, jsonData, 0644)
}

func (r *HeartbeatReceiver) ParseLocalFile(ctx context.Context) error {
	// Check if the file exists
	if _, err := os.Stat(localDataFilePath); err != nil {
		return err
	}

	// Read the file
	jsonData, err := os.ReadFile(localDataFilePath)
	if err != nil {
		return err
	}

	// Unmarshal the JSON data into localData
	return json.Unmarshal(jsonData, &localData)
}

func (r *HeartbeatReceiver) InitializeAgentLocalData(ctx context.Context) error {
	// Set the local start time
	localData.AgentStartTime = time.Now().Unix()

	// Parse the local file if it exists
	err := r.ParseLocalFile(ctx)

	// If the local file doesn't exist, then generate new local data and persist it to disk
	if err != nil && os.IsNotExist(err) {
		localData.AgentInstanceId = r.GenerateAgentInstanceId(ctx)
		// Persist the local data to file
		if err := r.PersistToLocalFile(ctx); err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}
