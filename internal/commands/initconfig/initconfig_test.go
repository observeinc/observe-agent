package initconfig

import (
	"os"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/go-defaults"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func setConfigDefaults(agentConfig config.AgentConfig) config.AgentConfig {
	defaults.SetDefaults(&agentConfig)
	return agentConfig
}

func Test_InitConfigCommand(t *testing.T) {
	t.Cleanup(func() {
		os.Remove("./test-config.yaml")
	})
	testcases := []struct {
		args           []string
		expectedConfig config.AgentConfig
		expectErr      string
	}{
		{
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--host_monitoring::logs::include=/test/path,/test/path2"},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				SelfMonitoring: config.SelfMonitoringConfig{
					Enabled: true,
					Fleet: config.FleetHeartbeatConfig{
						Enabled: true,
					},
				},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs: config.HostMonitoringLogsConfig{
						Enabled: true,
						Include: []string{"/test/path", "/test/path2"},
					},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{
							Enabled: true,
						},
						Process: config.HostMonitoringProcessMetricsConfig{
							Enabled: false,
						},
					},
				},
			}),
			expectErr: "",
		},
		{
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--self_monitoring::enabled=false", "--self_monitoring::fleet::enabled=false", "--host_monitoring::enabled=false", "--host_monitoring::logs::enabled=false", "--host_monitoring::metrics::host::enabled=false", "--host_monitoring::metrics::process::enabled=false"},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: false,
					Logs: config.HostMonitoringLogsConfig{
						Enabled: false,
					},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{
							Enabled: false,
						},
						Process: config.HostMonitoringProcessMetricsConfig{
							Enabled: false,
						},
					},
				},
			}),
			expectErr: "",
		},
		{
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--forwarding::metrics::output_format=otel"},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				Forwarding: config.ForwardingConfig{
					Metrics: config.ForwardingMetricsConfig{
						OutputFormat: "otel",
					},
				},
				SelfMonitoring: config.SelfMonitoringConfig{
					Enabled: true,
					Fleet: config.FleetHeartbeatConfig{
						Enabled: true,
					},
				},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs: config.HostMonitoringLogsConfig{
						Enabled: true,
					},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{
							Enabled: true,
						},
					},
				},
			}),
			expectErr: "",
		},
		{
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--self_monitoring::fleet::enabled=true", "--self_monitoring::fleet::interval=5m", "--self_monitoring::fleet::config_interval=30m"},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				SelfMonitoring: config.SelfMonitoringConfig{
					Enabled: true,
					Fleet: config.FleetHeartbeatConfig{
						Enabled:        true,
						Interval:       "5m",
						ConfigInterval: "30m",
					},
				},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs: config.HostMonitoringLogsConfig{
						Enabled: true,
					},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{
							Enabled: true,
						},
					},
				},
			}),
			expectErr: "",
		},
		// Exercise the newly-added flags from the init-config-flags PR. This
		// covers every section that previously had no CLI surface so the
		// matrix of "did we wire this up correctly?" stays answerable.
		//
		// Note: this test case only exercises fields whose default is the Go
		// zero value (false / "" / nil). Default-true bools (e.g.
		// internal_telemetry::*::enabled, health_check::enabled,
		// forwarding::enabled) need a different test setup because
		// setConfigDefaults — go-defaults — will re-apply schema defaults
		// over any false we set explicitly, so we can't assert that the
		// flag flipped them. They get their own test below.
		{
			args: []string{
				"--config_path=./test-config.yaml",
				"--token=test-token",
				"--observe_url=test-url",
				"--omit_base_components=true",
				"--agent_local_file_path=/var/lib/observe-agent",
				"--attributes=team=platform,deployment.environment=staging",
				"--application::RED_metrics::enabled=true",
				"--application::RED_metrics::only_generate_for_service_entrypoint_spans=true",
				"--application::RED_metrics::resource_dimensions=service.namespace,service.version",
				"--application::RED_metrics::span_dimensions=peer.db.name,otel.status_description",
				"--health_check::endpoint=0.0.0.0:13133",
				"--health_check::path=/healthz",
				"--forwarding::endpoints::http=0.0.0.0:4318",
				"--forwarding::endpoints::grpc=0.0.0.0:4317",
				"--forwarding::metrics::convert_cumulative_to_delta=true",
				"--forwarding::traces::max_span_duration=30m",
				"--internal_telemetry::metrics::host=0.0.0.0",
				"--internal_telemetry::metrics::port=9999",
				"--internal_telemetry::metrics::level=basic",
				"--internal_telemetry::logs::level=debug",
				"--internal_telemetry::logs::encoding=json",
				"--host_monitoring::logs::exclude=/var/log/btmp,/var/log/wtmp",
				"--exporters::sending_queue_batch::enabled=true",
				"--exporters::sending_queue_batch::max_size=1048576",
				"--exporters::emit_prometheus_target_info_metric=true",
			},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:              "test-token",
				ObserveURL:         "test-url",
				OmitBaseComponents: true,
				AgentLocalFilePath: "/var/lib/observe-agent",
				Attributes: map[string]string{
					"team":                   "platform",
					"deployment.environment": "staging",
				},
				Application: config.ApplicationConfig{
					REDMetrics: config.REDMetricsConfig{
						Enabled:                               true,
						OnlyGenerateForServiceEntrypointSpans: true,
						ResourceDimensions:                    []string{"service.namespace", "service.version"},
						SpanDimensions:                        []string{"peer.db.name", "otel.status_description"},
					},
				},
				HealthCheck: config.HealthCheckConfig{
					Endpoint: "0.0.0.0:13133",
					Path:     "/healthz",
				},
				Forwarding: config.ForwardingConfig{
					Endpoints: config.ForwardingReceiverEndpointsConfig{
						HTTP: "0.0.0.0:4318",
						GRPC: "0.0.0.0:4317",
					},
					Metrics: config.ForwardingMetricsConfig{
						ConvertCumulativeToDelta: true,
					},
					Traces: config.ForwardingTracesConfig{
						MaxSpanDuration: "30m",
					},
				},
				InternalTelemetry: config.InternalTelemetryConfig{
					Metrics: config.InternalTelemetryMetricsConfig{
						Host:  "0.0.0.0",
						Port:  9999,
						Level: "basic",
					},
					Logs: config.InternalTelemetryLogsConfig{
						Level:    "debug",
						Encoding: "json",
					},
				},
				SelfMonitoring: config.SelfMonitoringConfig{
					Enabled: true,
					Fleet: config.FleetHeartbeatConfig{
						Enabled: true,
					},
				},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs: config.HostMonitoringLogsConfig{
						Enabled: true,
						Exclude: []string{"/var/log/btmp", "/var/log/wtmp"},
					},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{
							Enabled: true,
						},
					},
				},
				Exporters: config.ExportersConfig{
					SendingQueueBatch: config.SendingQueueBatchConfig{
						Enabled: true,
						MaxSize: 1048576,
					},
					EmitPrometheusTargetInfoMetric: true,
				},
			}),
			expectErr: "",
		},
	}
	for _, tc := range testcases {
		v := viper.NewWithOptions(viper.KeyDelimiter("::"))
		config.SetViperDefaults(v, "::", "")
		initConfigCmd := NewConfigureCmd(v)
		RegisterConfigFlags(initConfigCmd, v)
		initConfigCmd.SetArgs(tc.args)
		err := initConfigCmd.Execute()
		if err != nil {
			if tc.expectErr == "" {
				t.Errorf("Expected no error, got %v", err)
			} else {
				assert.ErrorContains(t, err, tc.expectErr)
			}
		}

		var configYaml, configMapstructure config.AgentConfig

		// Decode via mapstructure (which is how viper does it)
		configFileContents, err := os.ReadFile("./test-config.yaml")
		assert.NoError(t, err)
		var yamlMap map[string]any
		err = yaml.Unmarshal(configFileContents, &yamlMap)
		assert.NoError(t, err)
		err = mapstructure.Decode(yamlMap, &configMapstructure)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedConfig, configMapstructure)

		// Decode via yaml in order to do strict field checks
		configFile, err := os.Open("./test-config.yaml")
		assert.NoError(t, err)
		decoder := yaml.NewDecoder(configFile)
		// Ensure that all fields in the output yaml are present in our struct
		decoder.KnownFields(true)
		err = decoder.Decode(&configYaml)
		assert.NoError(t, err)
		assert.Equal(t, tc.expectedConfig, configYaml)
	}
}

// Asserts that --flag=false flips the value for fields whose schema default is
// true. The main test above uses setConfigDefaults() to build expected configs,
// which calls go-defaults — and go-defaults treats `false` as a zero value and
// re-applies the schema's `default:"true"`, hiding the flip from the assertion.
// So we read the YAML directly and string-match here instead of round-tripping
// through AgentConfig.
func Test_InitConfigCommand_DefaultTrueBoolsCanBeFlippedToFalse(t *testing.T) {
	t.Cleanup(func() {
		os.Remove("./test-config.yaml")
	})

	args := []string{
		"--config_path=./test-config.yaml",
		"--token=test-token",
		"--observe_url=test-url",
		"--health_check::enabled=false",
		"--forwarding::enabled=false",
		"--internal_telemetry::enabled=false",
		"--internal_telemetry::metrics::enabled=false",
		"--internal_telemetry::logs::enabled=false",
	}

	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	config.SetViperDefaults(v, "::", "")
	cmd := NewConfigureCmd(v)
	RegisterConfigFlags(cmd, v)
	cmd.SetArgs(args)
	assert.NoError(t, cmd.Execute())

	contents, err := os.ReadFile("./test-config.yaml")
	assert.NoError(t, err)
	yamlStr := string(contents)

	// Each of these sections should contain the explicit `enabled: false` we
	// set via flag, not the schema default of true. Match scoped to each
	// section's heading so we don't false-positive on an `enabled: false`
	// from elsewhere in the file. Indentation-agnostic — yaml.Marshal can
	// emit 2- or 4-space; the writeConfigFile path currently emits 4.
	for _, section := range []string{
		"health_check",
		"forwarding",
		"internal_telemetry",
	} {
		assert.Regexp(t,
			`(?ms)^`+section+`:\n(.*\n)*?\s+enabled: false`,
			yamlStr,
			"%s.enabled should be flipped to false by --%s::enabled=false", section, section)
	}

	// internal_telemetry.metrics.enabled and .logs.enabled are nested another
	// level deeper.
	assert.Regexp(t,
		`(?ms)^internal_telemetry:\n(.*\n)*?\s+metrics:\n(.*\n)*?\s+enabled: false`,
		yamlStr,
		"internal_telemetry.metrics.enabled should be flipped to false")
	assert.Regexp(t,
		`(?ms)^internal_telemetry:\n(.*\n)*?\s+logs:\n(.*\n)*?\s+enabled: false`,
		yamlStr,
		"internal_telemetry.logs.enabled should be flipped to false")
}
