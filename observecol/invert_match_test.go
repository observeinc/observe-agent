package observecol

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/yaml.v3"
)

func confFromYAML(t *testing.T, content string) *confmap.Conf {
	var m map[string]any
	require.NoError(t, yaml.Unmarshal([]byte(content), &m))
	return confmap.NewFromStringMap(m)
}

func TestInvertMatchTailSamplingProcessors(t *testing.T) {
	tests := []struct {
		name string
		yaml string
		want []string
	}{
		{
			name: "top-level policy",
			yaml: `
processors:
  tail_sampling:
    policies:
      - name: exclude-health-checks
        type: string_attribute
        string_attribute: {key: http.target, values: [/healthz], invert_match: true}
`,
			want: []string{"tail_sampling"},
		},
		{
			name: "and sub-policy and string value",
			yaml: `
processors:
  tail_sampling/b:
    policies:
      - name: and
        type: and
        and:
          and_sub_policy:
            - name: not-debug
              type: boolean_attribute
              boolean_attribute: {key: debug, value: true, invert_match: "true"}
  tail_sampling/a:
    policies:
      - name: slow
        type: numeric_attribute
        numeric_attribute: {key: duration, min_value: 1, max_value: 2, invert_match: true}
`,
			want: []string{"tail_sampling/a", "tail_sampling/b"},
		},
		{
			name: "invert_match false",
			yaml: `
processors:
  tail_sampling:
    policies:
      - name: health-checks
        type: string_attribute
        string_attribute: {key: http.target, values: [/healthz], invert_match: false}
`,
		},
		{
			name: "other processor types",
			yaml: `
processors:
  filter/invert_match:
    invert_match: true
  tail_sampling:
    policies:
      - {name: errors, type: status_code, status_code: {status_codes: [ERROR]}}
`,
		},
		{
			name: "no processors",
			yaml: `receivers: {}`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, invertMatchTailSamplingProcessors(confFromYAML(t, tt.yaml)))
		})
	}
}

func TestInvertMatchConverterWarns(t *testing.T) {
	core, logs := observer.New(zapcore.WarnLevel)
	converter := newInvertMatchConverterFactory().Create(confmap.ConverterSettings{Logger: zap.New(core)})
	conf := confFromYAML(t, `
processors:
  tail_sampling/observe:
    policies:
      - name: exclude-health-checks
        type: string_attribute
        string_attribute: {key: http.target, values: [/healthz], invert_match: true}
`)
	require.NoError(t, converter.Convert(context.Background(), conf))

	entries := logs.All()
	require.Len(t, entries, 1)
	assert.Equal(t, InvertMatchWarning, entries[0].Message)
	assert.Equal(t, "tail_sampling/observe", entries[0].ContextMap()["processor"])
}
