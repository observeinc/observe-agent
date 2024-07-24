package observek8sattributesprocessor

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type observeTransformFacets struct {
	facets map[string]string
}

type logWithResource struct {
	logName            string
	resourceAttributes map[string]any
	recordAttributes   map[string]any
	severityText       string
	body               string
	testBodyFilepath   string
	severityNumber     plog.SeverityNumber
}

type K8sEventProcessorTest struct {
	name          string
	inLogs        plog.Logs
	outAttributes []map[string]interface{}
}

var (
	podStatusInLogs = []logWithResource{
		{
			logName:          "noObserveTransformAttributes",
			testBodyFilepath: "./testdata/podObjectEvent.json",
		},
		{
			logName:          "existingObserveTransformAttributes",
			testBodyFilepath: "./testdata/podObjectEvent.json",
			recordAttributes: map[string]any{
				"observe_transform": map[string]any{
					"facets": map[string]any{
						"other_key": "test",
					},
				},
			},
		},
	}
	podStatusOutAttributes = []map[string]interface{}{
		{
			"observe_transform": map[string]interface{}{
				"facets": map[string]interface{}{
					"status": "Terminating",
				},
			},
			"name": "noObserveTransformAttributes",
		},
		{
			"observe_transform": map[string]interface{}{
				"facets": map[string]interface{}{
					"status":    "Terminating",
					"other_key": "test",
				},
			},
			"name": "existingObserveTransformAttributes",
		},
	}

	tests = []K8sEventProcessorTest{
		{
			name:          "noObserveTransformAttributes",
			inLogs:        testResourceLogs(podStatusInLogs),
			outAttributes: podStatusOutAttributes,
		},
	}
)

func TestK8sEventsProcessor(t *testing.T) {
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			kep := newK8sEventsProcessor(zap.NewNop(), &Config{})
			logs, err := kep.processLogs(context.Background(), test.inLogs)
			for i := 0; i < logs.ResourceLogs().Len(); i++ {
				sl := logs.ResourceLogs().At(i).ScopeLogs()
				for j := 0; j < sl.Len(); j++ {
					lr := sl.At(j).LogRecords()
					require.Equal(t, test.outAttributes[i], lr.At(0).Attributes().AsRaw())
				}
			}

			require.NoError(t, err)
		})
	}
}

func testResourceLogs(lwrs []logWithResource) plog.Logs {
	ld := plog.NewLogs()

	for i, lwr := range lwrs {
		rl := ld.ResourceLogs().AppendEmpty()

		// Add resource level attributes
		//nolint:errcheck
		rl.Resource().Attributes().FromRaw(lwr.resourceAttributes)
		ls := rl.ScopeLogs().AppendEmpty().LogRecords()
		l := ls.AppendEmpty()
		// Add record level attributes
		//nolint:errcheck
		l.Attributes().FromRaw(lwrs[i].recordAttributes)
		l.Attributes().PutStr("name", lwr.logName)
		// Set body & severity fields
		if lwr.body != "" {
			l.Body().SetStr(lwr.body)
		} else if lwr.testBodyFilepath != "" {
			file, err := ioutil.ReadFile(lwr.testBodyFilepath)
			if err == nil {
				l.Body().SetStr(string(file))
			}
		}
		l.SetSeverityText(lwr.severityText)
		l.SetSeverityNumber(lwr.severityNumber)
	}
	return ld
}
