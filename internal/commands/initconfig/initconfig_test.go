package initconfig

import (
	"os"
	"reflect"
	"strings"
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
			args: []string{"--config_path=./test-config.yaml", "--token=test-token", "--observe_url=test-url", "--self_monitoring::enabled=false", "--host_monitoring::enabled=false", "--host_monitoring::logs::enabled=false", "--host_monitoring::metrics::host::enabled=false", "--host_monitoring::metrics::process::enabled=false"},
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
		// Newly-covered fields: forwarding endpoints
		{
			args: []string{
				"--config_path=./test-config.yaml",
				"--token=test-token",
				"--observe_url=test-url",
				"--forwarding::endpoints::http=0.0.0.0:4318",
				"--forwarding::endpoints::grpc=0.0.0.0:4317",
			},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				Forwarding: config.ForwardingConfig{
					Endpoints: config.ForwardingReceiverEndpointsConfig{
						HTTP: "0.0.0.0:4318",
						GRPC: "0.0.0.0:4317",
					},
				},
				SelfMonitoring: config.SelfMonitoringConfig{Enabled: true},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs:    config.HostMonitoringLogsConfig{Enabled: true},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{Enabled: true},
					},
				},
			}),
			expectErr: "",
		},
		// Newly-covered fields: internal telemetry
		{
			args: []string{
				"--config_path=./test-config.yaml",
				"--token=test-token",
				"--observe_url=test-url",
				"--internal_telemetry::metrics::port=9999",
				"--internal_telemetry::logs::encoding=json",
			},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				InternalTelemetry: config.InternalTelemetryConfig{
					Metrics: config.InternalTelemetryMetricsConfig{Port: 9999},
					Logs:    config.InternalTelemetryLogsConfig{Encoding: "json"},
				},
				SelfMonitoring: config.SelfMonitoringConfig{Enabled: true},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs:    config.HostMonitoringLogsConfig{Enabled: true},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{Enabled: true},
					},
				},
			}),
			expectErr: "",
		},
		// Newly-covered fields: health check endpoint and path
		{
			args: []string{
				"--config_path=./test-config.yaml",
				"--token=test-token",
				"--observe_url=test-url",
				"--health_check::endpoint=0.0.0.0:13133",
				"--health_check::path=/custom-status",
			},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				HealthCheck: config.HealthCheckConfig{
					Endpoint: "0.0.0.0:13133",
					Path:     "/custom-status",
				},
				SelfMonitoring: config.SelfMonitoringConfig{Enabled: true},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs:    config.HostMonitoringLogsConfig{Enabled: true},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{Enabled: true},
					},
				},
			}),
			expectErr: "",
		},
		// Newly-covered fields: exporters
		{
			args: []string{
				"--config_path=./test-config.yaml",
				"--token=test-token",
				"--observe_url=test-url",
				"--exporters::sending_queue_batch::max_size=1000",
				"--exporters::emit_prometheus_target_info_metric=true",
			},
			expectedConfig: setConfigDefaults(config.AgentConfig{
				Token:      "test-token",
				ObserveURL: "test-url",
				Exporters: config.ExportersConfig{
					SendingQueueBatch:              config.SendingQueueBatchConfig{MaxSize: 1000},
					EmitPrometheusTargetInfoMetric: true,
				},
				SelfMonitoring: config.SelfMonitoringConfig{Enabled: true},
				HostMonitoring: config.HostMonitoringConfig{
					Enabled: true,
					Logs:    config.HostMonitoringLogsConfig{Enabled: true},
					Metrics: config.HostMonitoringMetricsConfig{
						Host: config.HostMonitoringHostMetricsConfig{Enabled: true},
					},
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

// Test_AllConfigFieldsHaveFlags walks AgentConfig via reflection and asserts
// that every leaf field has a corresponding registered cobra flag (or is in the
// known skip set). This test will fail if a new field is added to AgentConfig
// without the reflection walker being able to handle it, making coverage
// machine-checked.
func Test_AllConfigFieldsHaveFlags(t *testing.T) {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	cmd := NewConfigureCmd(v)
	RegisterConfigFlags(cmd, v)

	skip := map[string]bool{
		"debug": true, // deprecated
	}

	var checkFields func(typ reflect.Type, prefix string)
	checkFields = func(typ reflect.Type, prefix string) {
		for i := 0; i < typ.NumField(); i++ {
			field := typ.Field(i)
			mapKey := strings.Split(field.Tag.Get("mapstructure"), ",")[0]
			if mapKey == "" || mapKey == "-" {
				continue
			}
			viperKey := mapKey
			if prefix != "" {
				viperKey = prefix + "::" + mapKey
			}
			if skip[viperKey] {
				continue
			}
			switch field.Type.Kind() {
			case reflect.Struct:
				checkFields(field.Type, viperKey)
			case reflect.Bool, reflect.String, reflect.Int:
				assert.NotNil(t, cmd.PersistentFlags().Lookup(viperKey),
					"missing flag for field: %s", viperKey)
			case reflect.Slice:
				if field.Type.Elem().Kind() == reflect.String {
					assert.NotNil(t, cmd.PersistentFlags().Lookup(viperKey),
						"missing flag for field: %s", viperKey)
				}
			case reflect.Map:
				if field.Type.Key().Kind() == reflect.String &&
					field.Type.Elem().Kind() == reflect.String {
					assert.NotNil(t, cmd.PersistentFlags().Lookup(viperKey),
						"missing flag for field: %s", viperKey)
				}
			}
		}
	}
	checkFields(reflect.TypeOf(config.AgentConfig{}), "")
}
