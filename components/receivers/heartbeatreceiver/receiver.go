package heartbeatreceiver

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/observeinc/observe-agent/components/receivers/heartbeatreceiver/internal/metadata"
	"github.com/observeinc/observe-agent/internal/connections"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
	"go.opentelemetry.io/collector/confmap/provider/envprovider"
	"go.opentelemetry.io/collector/confmap/provider/fileprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpprovider"
	"go.opentelemetry.io/collector/confmap/provider/httpsprovider"
	"go.opentelemetry.io/collector/confmap/provider/yamlprovider"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
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

	go func() {
		for {
			select {
			case <-r.ticker.C:
				if err := r.generateLifecycleHeartbeat(ctx); err != nil {
					r.settings.Logger.Error("failed to generate lifecycle heartbeat", zap.Error(err))
					// Continue - don't stop timer on error
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	// Start config heartbeat timer
	configInterval, _ := time.ParseDuration(r.cfg.ConfigInterval)
	r.configTicker = time.NewTicker(configInterval)
	r.settings.Logger.Info("Config heartbeat enabled",
		zap.String("interval", r.cfg.ConfigInterval))

	go func() {
		for {
			select {
			case <-r.configTicker.C:
				if err := r.generateConfigHeartbeat(ctx); err != nil {
					r.settings.Logger.Error("failed to generate config heartbeat", zap.Error(err))
					// Continue - don't stop timer on error
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

	// Control subobject
	control := observe_transform.PutEmptyMap("control")
	control.PutBool("isDelete", false)

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

	// Add lifecycle-specific body fields
	body := logRecord.Body().SetEmptyMap()
	body.PutStr("agent_instance_id", r.state.AgentInstanceId)
	body.PutInt("agent_start_time", r.state.AgentStartTime)

	// Add auth check results to the log body under a nested object
	authCheck := body.PutEmptyMap("auth_check")
	authCheck.PutBool("passed", authResult.Passed)
	authCheck.PutStr("url", authResult.URL)
	authCheck.PutInt("response_code", int64(authResult.ResponseCode))
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

// SensitiveFieldPattern defines a pattern for matching and obfuscating sensitive fields in YAML
type SensitiveFieldPattern struct {
	// Path is the YAML path to the field using dot notation
	// Examples: "token", "auth_check.headers.authorization", "database.password"
	Path string

	// PrefixLength is the number of characters to show before obfuscating (default: 8)
	PrefixLength int
}

// sensitiveFieldPatterns defines all the sensitive fields that should be obfuscated
var sensitiveFieldPatterns = []SensitiveFieldPattern{
	{
		Path:         "token",
		PrefixLength: 8,
	},
	// Add more patterns here as needed, e.g.:
	// {
	//     Path:         "auth_check.headers.authorization",
	//     PrefixLength: 8,
	// },
}

// obfuscateValue obfuscates a value by showing a prefix and replacing the rest with asterisks
func obfuscateValue(value string, prefixLength int) string {
	if len(value) > prefixLength {
		return value[:prefixLength] + strings.Repeat("*", len(value)-prefixLength)
	}
	return strings.Repeat("*", len(value))
}

// traverseAndObfuscate recursively traverses a YAML node and obfuscates sensitive fields
func traverseAndObfuscate(node *yaml.Node, currentPath []string, patterns []SensitiveFieldPattern) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// Document node contains a single content node
		if len(node.Content) > 0 {
			traverseAndObfuscate(node.Content[0], currentPath, patterns)
		}

	case yaml.MappingNode:
		// Mapping nodes have key-value pairs in Content
		// Content is a flat list: [key1, value1, key2, value2, ...]
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}

			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Build the path for this key
			newPath := append(currentPath, keyNode.Value)

			// Check if this path matches any sensitive field pattern
			for _, pattern := range patterns {
				// Split the pattern path by dots
				patternPath := strings.Split(pattern.Path, ".")

				if pathsMatch(newPath, patternPath) {
					// Obfuscate the value
					if valueNode.Kind == yaml.ScalarNode {
						prefixLen := pattern.PrefixLength
						if prefixLen == 0 {
							prefixLen = 8
						}
						valueNode.Value = obfuscateValue(valueNode.Value, prefixLen)
					}
				}
			}

			// Recurse into the value node
			traverseAndObfuscate(valueNode, newPath, patterns)
		}

	case yaml.SequenceNode:
		// Sequence nodes contain list items
		for _, item := range node.Content {
			traverseAndObfuscate(item, currentPath, patterns)
		}

	case yaml.ScalarNode:
		// Scalar nodes are leaf values - nothing to traverse
		return
	}
}

// pathsMatch checks if the current path matches the pattern path
func pathsMatch(current []string, pattern []string) bool {
	if len(current) != len(pattern) {
		return false
	}
	for i := range pattern {
		if current[i] != pattern[i] {
			return false
		}
	}
	return true
}

// obfuscateSensitiveFields replaces sensitive field values in YAML content with obfuscated versions
// This implementation preserves the original YAML formatting
func obfuscateSensitiveFields(content []byte) []byte {
	result := content

	// Apply each sensitive field pattern
	for _, pattern := range sensitiveFieldPatterns {
		prefixLen := pattern.PrefixLength
		if prefixLen == 0 {
			prefixLen = 8
		}

		// Use YAML parsing for accuracy
		var node yaml.Node
		err := yaml.Unmarshal(result, &node)
		if err != nil {
			continue
		}

		// Find and obfuscate the value at the path
		traverseAndObfuscate(&node, []string{}, []SensitiveFieldPattern{pattern})

		// Marshal back
		obfuscated, err := yaml.Marshal(&node)
		if err != nil {
			continue
		}

		result = obfuscated
	}

	return result
}

// getObserveAgentConfigBytes reads the observe-agent.yaml config file and returns it as base64 encoded string
func (r *HeartbeatReceiver) getObserveAgentConfigBytes(ctx context.Context) (string, error) {
	configFile := viper.ConfigFileUsed()
	if configFile == "" {
		return "", fmt.Errorf("no config file in use")
	}

	content, err := os.ReadFile(configFile)
	if err != nil {
		r.settings.Logger.Error("failed to read observe-agent config",
			zap.String("configFile", configFile), zap.Error(err))
		return "", err
	}

	// Obfuscate sensitive fields before encoding
	obfuscatedContent := obfuscateSensitiveFields(content)

	encoded := base64.StdEncoding.EncodeToString(obfuscatedContent)
	return encoded, nil
}

// getOtelConfigBytes generates the fully merged OTEL config and returns it as base64 encoded string
func (r *HeartbeatReceiver) getOtelConfigBytes(ctx context.Context) (string, error) {
	// Get config files and setup collector settings
	// We replicate the logic from PrintFullOtelConfig here to avoid import cycle
	configFiles, cleanup, err := connections.SetupAndGetConfigFiles(ctx)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		r.settings.Logger.Error("failed to setup and get config files", zap.Error(err))
		return "", err
	}

	// Get the config provider settings using the same approach as GetOtelCollectorSettings
	URIs := configFiles
	if len(URIs) == 0 {
		return "", fmt.Errorf("no config URIs available")
	}

	// Create a resolver to get the merged config
	resolverSettings := confmap.ResolverSettings{
		URIs:          URIs,
		DefaultScheme: "env",
		ProviderFactories: []confmap.ProviderFactory{
			fileprovider.NewFactory(),
			envprovider.NewFactory(),
			yamlprovider.NewFactory(),
			httpprovider.NewFactory(),
			httpsprovider.NewFactory(),
		},
	}

	resolver, err := confmap.NewResolver(resolverSettings)
	if err != nil {
		r.settings.Logger.Error("failed to create resolver", zap.Error(err))
		return "", err
	}

	conf, err := resolver.Resolve(ctx)
	if err != nil {
		r.settings.Logger.Error("failed to resolve config", zap.Error(err))
		return "", err
	}

	// Convert to YAML
	cfgYaml, err := yaml.Marshal(conf.ToStringMap())
	if err != nil {
		r.settings.Logger.Error("failed to marshal config to YAML", zap.Error(err))
		return "", err
	}

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(cfgYaml)
	return encoded, nil
}

// generateConfigHeartbeat creates and sends a config heartbeat event
func (r *HeartbeatReceiver) generateConfigHeartbeat(ctx context.Context) error {
	r.settings.Logger.Debug("Generating config heartbeat",
		zap.String("agent_instance_id", r.state.AgentInstanceId))

	// Get base64 encoded configs
	agentConfig, err := r.getObserveAgentConfigBytes(ctx)
	if err != nil {
		r.settings.Logger.Error("failed to get observe-agent config bytes", zap.Error(err))
		return err
	}

	otelConfig, err := r.getOtelConfigBytes(ctx)
	if err != nil {
		r.settings.Logger.Error("failed to get OTEL config bytes", zap.Error(err))
		return err
	}

	// Create log entry
	logs := plog.NewLogs()
	resourceLogs := logs.ResourceLogs().AppendEmpty()
	scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
	logRecord := scopeLogs.LogRecords().AppendEmpty()

	// Add common heartbeat fields
	r.addCommonHeartbeatFields(resourceLogs, logRecord, "AgentConfig")

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
