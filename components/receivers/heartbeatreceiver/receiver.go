package heartbeatreceiver

import (
	"context"
	"fmt"
	"os"
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
				resourceLogs.Resource().Attributes().PutStr("observe.agent.instance.id", localData.AgentInstanceId)

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

func (r *HeartbeatReceiver) InitializeAgentLocalData(ctx context.Context) error {
	// Set the local start time
	localData.AgentStartTime = time.Now().Unix()
	localData.AgentInstanceId = os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	if localData.AgentInstanceId == "" {
		return fmt.Errorf("OBSERVE_AGENT_INSTANCE_ID environment variable must be set")
	}
	return nil
}
