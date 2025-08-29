package heartbeatreceiver

import (
	"context"
	"fmt"
	"os"
	"strconv"
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

type AuthCheckData struct {
	Passed       bool   `json:"passed"`
	URL          string `json:"url"`
	ResponseCode int    `json:"response_code"`
	Error        string `json:"error,omitempty"`
}

type HeartbeatLogRecord struct {
	AgentInstanceId string        `json:"agent_instance_id"`
	AgentStartTime  int64         `json:"agent_start_time"`
	AuthCheck       AuthCheckData `json:"auth_check"`
}

var localData HeartbeatLocalData

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
				// Perform authentication check
				authResult := PerformAuthCheck()

				r.settings.Logger.Info("Sending heartbeat",
					zap.String("agent_instance_id", localData.AgentInstanceId),
					zap.Bool("auth_check_passed", authResult.Passed),
					zap.String("auth_check_url", authResult.URL))

				logs := plog.NewLogs()
				resourceLogs := logs.ResourceLogs().AppendEmpty()
				resourceLogs.Resource().Attributes().PutStr("observe.agent.instance.id", localData.AgentInstanceId)
				resourceLogs.Resource().Attributes().PutStr("observe.agent.environment", r.cfg.Environment)
				resourceLogs.Resource().Attributes().PutStr("observe.agent.processId", strconv.Itoa(os.Getpid()))

				scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
				logRecord := scopeLogs.LogRecords().AppendEmpty()
				observe_transform := logRecord.Attributes().PutEmptyMap("observe_transform")

				// Identifiers subobject
				identifiers := observe_transform.PutEmptyMap("identifiers")
				identifiers.PutStr("agent_instance_id", localData.AgentInstanceId)

				// Control subobject
				control := observe_transform.PutEmptyMap("control")
				control.PutBool("isDelete", false)

				// observe_transform fields
				observe_transform.PutInt("process_start_time", localData.AgentStartTime)
				observe_transform.PutInt("valid_from", time.Now().UnixNano())
				observe_transform.PutInt("valid_to", time.Now().UnixNano()+5400000000000)
				observe_transform.PutStr("kind", "AgentLifecycleEvent")
				body := logRecord.Body().SetEmptyMap()
				body.PutStr("agent_instance_id", localData.AgentInstanceId)
				body.PutInt("agent_start_time", localData.AgentStartTime)

				// Add auth check results to the log body under a nested object
				authCheck := body.PutEmptyMap("auth_check")
				authCheck.PutBool("passed", authResult.Passed)
				authCheck.PutStr("url", authResult.URL)
				authCheck.PutInt("response_code", int64(authResult.ResponseCode))
				if authResult.Error != "" {
					authCheck.PutStr("error", authResult.Error)
				}

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

func (r *HeartbeatReceiver) InitializeAgentLocalData(ctx context.Context) error {
	// Set the local start time
	localData.AgentStartTime = time.Now().Unix()
	localData.AgentInstanceId = os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	if localData.AgentInstanceId == "" {
		return fmt.Errorf("OBSERVE_AGENT_INSTANCE_ID environment variable must be set")
	}
	return nil
}
