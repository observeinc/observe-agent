package config

import (
	"os"
	"reflect"
	"testing"

	"github.com/go-viper/mapstructure/v2"
	"github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentConfigValidate(t *testing.T) {
	validConfig := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
	}
	defaults.SetDefaults(&validConfig)
	assert.NoError(t, validConfig.Validate())

	validConfigOtel := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
		Forwarding: ForwardingConfig{
			Metrics: ForwardingMetricsConfig{
				OutputFormat: "otel",
			},
		},
	}
	defaults.SetDefaults(&validConfigOtel)
	assert.NoError(t, validConfigOtel.Validate())

	missingURLConfig := AgentConfig{
		Token:      "some:token",
		ObserveURL: "",
	}
	defaults.SetDefaults(&missingURLConfig)
	assert.ErrorContains(t, missingURLConfig.Validate(), "missing ObserveURL")

	invalidURLConfig1 := AgentConfig{
		Token:      "some:token",
		ObserveURL: "observeinc.com",
	}
	defaults.SetDefaults(&invalidURLConfig1)
	assert.ErrorContains(t, invalidURLConfig1.Validate(), "missing scheme for ObserveURL")

	invalidURLConfig2 := AgentConfig{
		Token:      "some:token",
		ObserveURL: "http://",
	}
	defaults.SetDefaults(&invalidURLConfig2)
	assert.ErrorContains(t, invalidURLConfig2.Validate(), "missing host for ObserveURL")

	missingTokenConfig := AgentConfig{
		Token:      "",
		ObserveURL: "https://observeinc.com",
	}
	defaults.SetDefaults(&missingTokenConfig)
	assert.ErrorContains(t, missingTokenConfig.Validate(), "missing Token")

	invalidTokenConfig := AgentConfig{
		Token:      "1234",
		ObserveURL: "https://observeinc.com",
	}
	defaults.SetDefaults(&invalidTokenConfig)
	assert.ErrorContains(t, invalidTokenConfig.Validate(), "invalid Token")

	invalidMetricsForwardingFormat := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
		Forwarding: ForwardingConfig{
			Enabled: true,
			Metrics: ForwardingMetricsConfig{
				OutputFormat: "invalid",
			},
		},
	}
	defaults.SetDefaults(&invalidMetricsForwardingFormat)
	assert.ErrorContains(t, invalidMetricsForwardingFormat.Validate(), "invalid metrics forwarding output format")

	emptyMetricsForwardingFormat := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
	}
	defaults.SetDefaults(&emptyMetricsForwardingFormat)
	emptyMetricsForwardingFormat.Forwarding.Metrics.OutputFormat = ""
	assert.ErrorContains(t, emptyMetricsForwardingFormat.Validate(), "invalid metrics forwarding output format")

	invalidMaxSpanDurationFormat := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://observeinc.com",
		Forwarding: ForwardingConfig{
			Enabled: true,
			Traces: ForwardingTracesConfig{
				MaxSpanDuration: "five months",
			},
		},
	}
	defaults.SetDefaults(&invalidMaxSpanDurationFormat)
	assert.ErrorContains(t, invalidMaxSpanDurationFormat.Validate(), "invalid max span duration 'five months' - Expected a number with a valid time unit: https://github.com/open-telemetry/opentelemetry-collector-contrib/blob/main/pkg/ottl/ottlfuncs/README.md#duration")

	os.Setenv("OBSERVE_AGENT_TEST_ID", "123")
	testEnvVarConfig := AgentConfig{
		Token:      "some:token",
		ObserveURL: "https://${env:OBSERVE_AGENT_TEST_ID}.collect.observeinc.com",
	}
	defaults.SetDefaults(&testEnvVarConfig)
	assert.NoError(t, testEnvVarConfig.Validate())
}

func TestStringToStringMapHookFunc(t *testing.T) {
	hook := StringToStringMapHookFunc()
	mapType := reflect.TypeOf(map[string]string{})
	strType := reflect.TypeOf("")

	cases := []struct {
		name    string
		from    reflect.Type
		to      reflect.Type
		input   any
		want    any
		wantErr string
	}{
		{
			name:  "pass-through when source is not a string",
			from:  mapType,
			to:    mapType,
			input: map[string]string{"k": "v"},
			want:  map[string]string{"k": "v"},
		},
		{
			name:  "pass-through when target is not map[string]string",
			from:  strType,
			to:    strType,
			input: "k=v",
			want:  "k=v",
		},
		{
			name:  "empty string returns empty map",
			from:  strType,
			to:    mapType,
			input: "",
			want:  map[string]string{},
		},
		{
			name:  "JSON format",
			from:  strType,
			to:    mapType,
			input: `{"service.name":"myservice","env":"prod"}`,
			want:  map[string]string{"service.name": "myservice", "env": "prod"},
		},
		{
			name:  "key=value pairs",
			from:  strType,
			to:    mapType,
			input: "service.name=myservice,env=prod",
			want:  map[string]string{"service.name": "myservice", "env": "prod"},
		},
		{
			name:  "single key=value pair",
			from:  strType,
			to:    mapType,
			input: "service.name=myservice",
			want:  map[string]string{"service.name": "myservice"},
		},
		{
			name:  "value containing equals sign",
			from:  strType,
			to:    mapType,
			input: "url=http://host/path?a=1",
			want:  map[string]string{"url": "http://host/path?a=1"},
		},
		{
			name:    "malformed pair returns error",
			from:    strType,
			to:      mapType,
			input:   "badvalue",
			wantErr: "invalid key=value pair",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fn, ok := hook.(mapstructure.DecodeHookFuncType)
			require.True(t, ok)
			got, err := fn(tc.from, tc.to, tc.input)
			if tc.wantErr != "" {
				assert.ErrorContains(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tc.want, got)
			}
		})
	}
}

func TestAgentConfigFromViper(t *testing.T) {
	config, err := AgentConfigFromViper(nil)
	assert.Error(t, err, "no viper instance provided")
	assert.Nil(t, config)

	type viperCase struct {
		name       string
		options    map[string]any
		configMode string
		validate   func(t *testing.T, config *AgentConfig)
	}

	cases := []viperCase{
		{
			name: "set values stick and unset values fall back to defaults",
			options: map[string]any{
				"token":                    "some:token",
				"observe_url":              "https://observeinc.com",
				"host_monitoring::enabled": true,
				"resource_attributes":      map[string]string{"deployment.environment.name": "test"},
			},
			configMode: "linux",
			validate: func(t *testing.T, config *AgentConfig) {
				assert.Equal(t, "some:token", config.Token)
				assert.Equal(t, "https://observeinc.com", config.ObserveURL)
				assert.Equal(t, true, config.HostMonitoring.Enabled)
				assert.Equal(t, "test", config.ResourceAttributes["deployment.environment.name"])

				// Defaults should be present when the value is not in the viper config.
				assert.Equal(t, true, config.HealthCheck.Enabled)
				assert.Equal(t, true, config.Forwarding.Enabled)
			},
		},
		{
			name: "resource_attributes from string simulates env var delivery",
			options: map[string]any{
				"token":               "some:token",
				"observe_url":         "https://observeinc.com",
				"resource_attributes": "service.name=myservice,env=prod",
			},
			configMode: "linux",
			validate: func(t *testing.T, config *AgentConfig) {
				assert.Equal(t, "myservice", config.ResourceAttributes["service.name"])
				assert.Equal(t, "prod", config.ResourceAttributes["env"])
			},
		},
		{
			name: "resource_attributes from JSON string simulates env var delivery",
			options: map[string]any{
				"token":               "some:token",
				"observe_url":         "https://observeinc.com",
				"resource_attributes": `{"service.name":"myservice","env":"prod"}`,
			},
			configMode: "linux",
			validate: func(t *testing.T, config *AgentConfig) {
				assert.Equal(t, "myservice", config.ResourceAttributes["service.name"])
				assert.Equal(t, "prod", config.ResourceAttributes["env"])
			},
		},
		{
			name: "user-supplied RED_metrics resource_dimensions in docker mode are not augmented with container.id",
			options: map[string]any{
				"token":                             "some:token",
				"observe_url":                       "https://observeinc.com",
				"application::RED_metrics::enabled": true,
				"application::RED_metrics::resource_dimensions": []string{"service.namespace", "service.version"},
			},
			configMode: "docker",
			validate: func(t *testing.T, config *AgentConfig) {
				assert.Equal(t, true, config.Application.REDMetrics.Enabled)
				assert.Equal(t, []string{"service.namespace", "service.version"}, config.Application.REDMetrics.ResourceDimensions)
				assert.NotContains(t, config.Application.REDMetrics.ResourceDimensions, "container.id")
				assert.NotContains(t, config.Application.REDMetrics.ResourceDimensions, "host.name")

				// OnlyGenerateForAPMSpans should default to true.
				assert.Equal(t, true, config.Application.REDMetrics.OnlyGenerateForAPMSpans)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			v := viper.NewWithOptions(viper.KeyDelimiter("::"))
			SetViperDefaults(v, "::", tc.configMode)
			for k, val := range tc.options {
				v.Set(k, val)
			}
			config, err := AgentConfigFromViper(v)
			assert.NoError(t, err)
			tc.validate(t, config)
		})
	}
}

func TestSetViperDefaultsDockerConfigMode(t *testing.T) {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	SetViperDefaults(v, "::", "docker")

	assert.Equal(t, "0.0.0.0:4318", v.GetString("forwarding::endpoints::http"))
	assert.Equal(t, "0.0.0.0:4317", v.GetString("forwarding::endpoints::grpc"))
	assert.Equal(t, "0.0.0.0:13133", v.GetString("health_check::endpoint"))
	assert.Equal(t, "0.0.0.0", v.GetString("internal_telemetry::metrics::host"))
	assert.Contains(t, v.GetStringSlice("application::RED_metrics::resource_dimensions"), "container.id")
	assert.Contains(t, v.GetStringSlice("application::RED_metrics::resource_dimensions"), "host.name")
}

func TestSetViperDefaultsNonDockerConfigMode(t *testing.T) {
	v := viper.NewWithOptions(viper.KeyDelimiter("::"))
	SetViperDefaults(v, "::", "linux")

	assert.Equal(t, "localhost:4318", v.GetString("forwarding::endpoints::http"))
	assert.Equal(t, "localhost:4317", v.GetString("forwarding::endpoints::grpc"))
	assert.Equal(t, "localhost:13133", v.GetString("health_check::endpoint"))
	assert.Equal(t, "localhost", v.GetString("internal_telemetry::metrics::host"))
	assert.Contains(t, v.GetStringSlice("application::RED_metrics::resource_dimensions"), "host.name")
	assert.NotContains(t, v.GetStringSlice("application::RED_metrics::resource_dimensions"), "container.id")
}
