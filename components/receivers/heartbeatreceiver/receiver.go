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
	"go.opentelemetry.io/collector/pdata/pcommon"
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
	configTicker *time.Ticker
	cancel       context.CancelFunc
	state        HeartbeatReceiverState
}

type HeartbeatReceiverState struct {
	AgentInstanceId string `json:"agent_instance_id"`
	AgentStartTime  int64
	AgentVersion    string `json:"agent_version"`
	Hostname        string `json:"hostname"`
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

type ConfigHeartbeatLogRecord struct {
	ObserveAgentConfig string `json:"observeAgentConfig"`
	OtelConfig         string `json:"otelConfig"`
}

func newReceiver(set receiver.Settings, cfg *Config, consumer consumer.Logs) (*HeartbeatReceiver, error) {
	// Initialize sensitive field patterns for redaction
	initSensitiveFieldPatterns()

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
	err := r.InitializeReceiverState(ctx)
	if err != nil {
		return err
	}

	// Start lifecycle heartbeat timer
	interval, _ := time.ParseDuration(r.cfg.Interval)
	r.ticker = time.NewTicker(interval)

	// Start config heartbeat timer
	configInterval, _ := time.ParseDuration(r.cfg.ConfigInterval)
	r.configTicker = time.NewTicker(configInterval)
	r.settings.Logger.Info("Config heartbeat enabled",
		zap.String("interval", r.cfg.ConfigInterval))

	swallowGenerateLifecycleHeartbeat := func() {
		if err := r.generateLifecycleHeartbeat(ctx); err != nil {
			r.settings.Logger.Error("failed to generate lifecycle heartbeat", zap.Error(err))
			// Continue to start the timer
		}
	}
	swallowGenerateConfigHeartbeat := func() {
		if err := r.generateConfigHeartbeat(ctx); err != nil {
			r.settings.Logger.Error("failed to generate config heartbeat", zap.Error(err))
			// Continue to start the timer
		}
	}
	go func() {
		// Generate initial heartbeats
		swallowGenerateLifecycleHeartbeat()
		swallowGenerateConfigHeartbeat()

		for {
			select {
			case <-r.ticker.C:
				swallowGenerateLifecycleHeartbeat()
			case <-r.configTicker.C:
				swallowGenerateConfigHeartbeat()
			case <-ctx.Done():
				return
			}
		}
	}()

	return nil
}

func (r *HeartbeatReceiver) Shutdown(ctx context.Context) error {
	r.settings.Logger.Info("Shutting down heartbeat receiver")

	// Send shutdown heartbeat before stopping tickers
	if err := r.generateShutdownHeartbeat(ctx); err != nil {
		r.settings.Logger.Error("failed to generate shutdown heartbeat", zap.Error(err))
		// Continue with shutdown even if heartbeat fails
	}

	if r.ticker != nil {
		r.ticker.Stop()
	}
	if r.configTicker != nil {
		r.configTicker.Stop()
	}
	if r.cancel != nil {
		r.cancel()
	}

	return nil
}

func (r *HeartbeatReceiver) InitializeReceiverState(ctx context.Context) error {
	// Set the local start time
	r.state.AgentStartTime = time.Now().Unix()
	r.state.AgentInstanceId = os.Getenv("OBSERVE_AGENT_INSTANCE_ID")
	if r.state.AgentInstanceId == "" {
		return fmt.Errorf("OBSERVE_AGENT_INSTANCE_ID environment variable must be set")
	}
	r.state.AgentVersion = os.Getenv("OBSERVE_AGENT_VERSION")
	if r.state.AgentVersion == "" {
		r.settings.Logger.Error("OBSERVE_AGENT_VERSION environment variable is not set")
	}
	hostname, err := os.Hostname()
	if err != nil {
		r.settings.Logger.Error("failed to get hostname", zap.Error(err))
	} else {
		r.state.Hostname = hostname
	}
	return nil
}

// addCommonHeartbeatFields adds the common resource attributes and observe_transform fields to a log record
func (r *HeartbeatReceiver) addCommonHeartbeatFields(resourceLogs plog.ResourceLogs, logRecord plog.LogRecord, kind string) {
	// Add resource attributes
	resourceLogs.Resource().Attributes().PutStr("observe.agent.instance.id", r.state.AgentInstanceId)
	resourceLogs.Resource().Attributes().PutStr("observe.agent.environment", r.cfg.Environment)
	resourceLogs.Resource().Attributes().PutStr("observe.agent.processId", strconv.Itoa(os.Getpid()))

	// Add fields to the observe_transform object
	observe_transform := logRecord.Attributes().PutEmptyMap("observe_transform")

	// Identifiers subobject
	identifiers := observe_transform.PutEmptyMap("identifiers")
	identifiers.PutStr("observe.agent.instance.id", r.state.AgentInstanceId)
	identifiers.PutStr("host.name", r.state.Hostname)
	identifiers.PutStr("observe.agent.environment", r.cfg.Environment)

	// facets subobject
	facets := observe_transform.PutEmptyMap("facets")
	facets.PutStr("observe.agent.version", r.state.AgentVersion)

	// observe_transform fields
	observe_transform.PutInt("process_start_time", r.state.AgentStartTime)
	observe_transform.PutInt("valid_from", time.Now().UnixNano())
	// The entities will be valid for 90 minutes
	observe_transform.PutInt("valid_to", time.Now().UnixNano()+5400000000000)
	observe_transform.PutStr("kind", kind)
}

// generateLifecycleHeartbeat creates and sends a lifecycle heartbeat event
func (r *HeartbeatReceiver) generateLifecycleHeartbeat(ctx context.Context) error {
	// Perform authentication check
	r.settings.Logger.Debug("Performing authentication check", zap.String("url", r.cfg.AuthCheck.URL))
	authResult := PerformAuthCheck(r.cfg.AuthCheck.URL, r.cfg.AuthCheck.Headers.Authorization)

	r.settings.Logger.Info("Sending lifecycle heartbeat",
		zap.String("agent_instance_id", r.state.AgentInstanceId),
		zap.Bool("auth_check_passed", authResult.Passed),
		zap.String("auth_check_url", authResult.URL))

	// Create log entry
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Add common heartbeat fields
	r.addCommonHeartbeatFields(resourceLogs, logRecord, "AgentLifecycleEvent")

	// Add control subobject to observe_transform which should have been added by addCommonHeartbeatFields
	var observe_transform pcommon.Map
	observeTransformValue, exists := logRecord.Attributes().Get("observe_transform")
	if !exists {
		observe_transform = logRecord.Attributes().PutEmptyMap("observe_transform")
	} else {
		if observeTransformValue.Type() != pcommon.ValueTypeMap {
			return fmt.Errorf("observe_transform attribute is not a map")
		}
		observe_transform = observeTransformValue.Map()
	}
	controlMap := observe_transform.PutEmptyMap("control")
	controlMap.PutStr("eventType", "HEARTBEAT")
	controlMap.PutBool("isDelete", false)

	// Add lifecycle-specific body fields
	body := logRecord.Body().SetEmptyMap()
	body.PutStr("agentInstanceId", r.state.AgentInstanceId)
	body.PutInt("agentStartTime", r.state.AgentStartTime)

	// Add auth check results to the log body under a nested object
	authCheck := body.PutEmptyMap("authCheck")
	authCheck.PutBool("passed", authResult.Passed)
	authCheck.PutStr("url", authResult.URL)
	authCheck.PutInt("responseCode", int64(authResult.ResponseCode))
	if authResult.Error != "" {
		authCheck.PutStr("error", authResult.Error)
	}

	// Send the log
	obsCtx := r.obsrecv.StartLogsOp(ctx)
	err := r.nextConsumer.ConsumeLogs(ctx, logs)
	r.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), 1, err)
	if err != nil {
		r.settings.Logger.Error("failed to send lifecycle heartbeat logs", zap.Error(err))
		return err
	}

	return nil
}

// generateConfigHeartbeat creates and sends a config heartbeat event
func (r *HeartbeatReceiver) generateConfigHeartbeat(ctx context.Context) error {
	r.settings.Logger.Debug("Generating config heartbeat",
		zap.String("agent_instance_id", r.state.AgentInstanceId))

	// Get configs from environment variables (base64 encoded)
	agentConfigEncoded := os.Getenv("OBSERVE_AGENT_CONFIG")
	otelConfigEncoded := os.Getenv("OBSERVE_AGENT_OTEL_CONFIG")

	if agentConfigEncoded == "" {
		r.settings.Logger.Error("OBSERVE_AGENT_CONFIG environment variable is not set")
	}

	if otelConfigEncoded == "" {
		r.settings.Logger.Error("OBSERVE_AGENT_OTEL_CONFIG environment variable is not set")
	}

	if agentConfigEncoded == "" && otelConfigEncoded == "" {
		r.settings.Logger.Error("Both OBSERVE_AGENT_CONFIG and OBSERVE_AGENT_OTEL_CONFIG were not set, skipping heartbeat")
		return nil // Don't crash, just skip this heartbeat
	}

	// Decode from base64
	agentConfigYaml, err := decodeBase64Config(agentConfigEncoded)
	if err != nil {
		r.settings.Logger.Error("failed to decode OBSERVE_AGENT_CONFIG from base64", zap.Error(err))
		agentConfigYaml = "" // Continue with empty config
	}

	otelConfigYaml, err := decodeBase64Config(otelConfigEncoded)
	if err != nil {
		r.settings.Logger.Error("failed to decode OBSERVE_AGENT_OTEL_CONFIG from base64", zap.Error(err))
		otelConfigYaml = "" // Continue with empty config
	}

	// Redact and encode configs
	agentConfig, err := redactAndEncodeConfig(agentConfigYaml)
	if err != nil {
		r.settings.Logger.Error("failed to redact and encode observe-agent config", zap.Error(err))
	}

	otelConfig, err := redactAndEncodeConfig(otelConfigYaml)
	if err != nil {
		r.settings.Logger.Error("failed to redact and encode OTEL config", zap.Error(err))
	}

	if agentConfig == "" && otelConfig == "" {
		r.settings.Logger.Error("Both config redaction processes failed, skipping heartbeat")
		return nil // Don't crash, just skip this heartbeat
	}

	// Create log entry
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Add common heartbeat fields
	r.addCommonHeartbeatFields(resourceLogs, logRecord, "AgentConfig")

	// Add control subobject to observe_transform which should have been added by addCommonHeartbeatFields
	var observe_transform pcommon.Map
	observeTransformValue, exists := logRecord.Attributes().Get("observe_transform")
	if !exists {
		observe_transform = logRecord.Attributes().PutEmptyMap("observe_transform")
	} else {
		if observeTransformValue.Type() != pcommon.ValueTypeMap {
			return fmt.Errorf("observe_transform attribute is not a map")
		}
		observe_transform = observeTransformValue.Map()
	}
	controlMap := observe_transform.PutEmptyMap("control")
	controlMap.PutStr("eventType", "CONFIG")
	controlMap.PutBool("isDelete", false)

	// Add config-specific body fields
	body := logRecord.Body().SetEmptyMap()
	body.PutStr("observeAgentConfig", agentConfig)
	body.PutStr("otelConfig", otelConfig)

	r.settings.Logger.Info("Sending config heartbeat",
		zap.String("agent_instance_id", r.state.AgentInstanceId))

	// Send the log
	obsCtx := r.obsrecv.StartLogsOp(ctx)
	err = r.nextConsumer.ConsumeLogs(ctx, logs)
	r.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), 1, err)
	if err != nil {
		r.settings.Logger.Error("failed to send config heartbeat logs", zap.Error(err))
		return err
	}

	return nil
}

// generateShutdownHeartbeat creates and sends a shutdown heartbeat event
func (r *HeartbeatReceiver) generateShutdownHeartbeat(ctx context.Context) error {
	r.settings.Logger.Info("Sending shutdown heartbeat",
		zap.String("agent_instance_id", r.state.AgentInstanceId))

	// Create log entry
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Add common heartbeat fields
	r.addCommonHeartbeatFields(resourceLogs, logRecord, "AgentLifecycleEvent")

	// Add control subobject to observe_transform
	var observe_transform pcommon.Map
	observeTransformValue, exists := logRecord.Attributes().Get("observe_transform")
	if !exists {
		observe_transform = logRecord.Attributes().PutEmptyMap("observe_transform")
	} else {
		if observeTransformValue.Type() != pcommon.ValueTypeMap {
			return fmt.Errorf("observe_transform attribute is not a map")
		}
		observe_transform = observeTransformValue.Map()
	}
	controlMap := observe_transform.PutEmptyMap("control")
	controlMap.PutStr("eventType", "SHUTDOWN")
	controlMap.PutBool("isDelete", true)

	// Add shutdown-specific body fields
	body := logRecord.Body().SetEmptyMap()
	body.PutStr("agentInstanceId", r.state.AgentInstanceId)
	body.PutInt("agentStartTime", r.state.AgentStartTime)

	// Send the log
	obsCtx := r.obsrecv.StartLogsOp(ctx)
	err := r.nextConsumer.ConsumeLogs(ctx, logs)
	r.obsrecv.EndLogsOp(obsCtx, metadata.Type.String(), 1, err)
	if err != nil {
		r.settings.Logger.Error("failed to send shutdown heartbeat logs", zap.Error(err))
		return err
	}

	return nil
}
