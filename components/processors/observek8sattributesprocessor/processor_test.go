package observek8sattributesprocessor

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jmespath/go-jmespath"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

// TODO [eg] refactor the testing infrastructure

type logWithResource struct {
	logName            string
	resourceAttributes map[string]any
	// These attributes simulate the enrichments performed on top of the raw
	// event by the pipelines in the agent config (generated from the template
	// in the helm repo)
	recordAttributes map[string]any
	severityText     string
	body             string
	testBodyFilepath string
	severityNumber   plog.SeverityNumber
}

// models a jmespath query against the processed results
type queryWithResult struct {
	path      string
	expResult any
}

type k8sEventProcessorTest struct {
	name            string
	inLogs          plog.Logs
	expectedResults []queryWithResult
}

func TestK8sEventsProcessor(t *testing.T) {
	for _, test := range []k8sEventProcessorTest{
		{
			name: "noObserveTransformAttributes",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEvent.json",
				},
			),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Terminating"},
			},
		},
		{ // Tests that we don't override/drop other facets computed in OTTL
			name: "existingObserveTransformAttributes",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEvent.json",
					recordAttributes: map[string]any{
						"observe_transform": map[string]interface{}{
							"facets": map[string]interface{}{
								"other_key": "test",
							},
						},
						"name": "existingObserveTransformAttributes",
					},
				},
			),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Terminating"},
				{"observe_transform.facets.other_key", "test"},
			},
		},
		{
			name: "Node Status and Role",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventSimple.json",
				},
			),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Ready"},
				{"observe_transform.facets.roles | length(@)", float64(1)},
				{"observe_transform.facets.roles[0]", "control-plane"},
			},
		},
		{
			name: "Node Status and Multiple Roles",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventAlternativeRoleKey.json",
				},
			),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Ready"},
				{"observe_transform.facets.roles | length(@)", float64(2)},
				{"observe_transform.facets.roles", []any{"anotherRole!", "control-plane"}},
			},
		},
		{
			name: "Pod container counts",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEvent.json",
				},
			),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.restarts", int64(5)},
				{"observe_transform.facets.total_containers", int64(4)},
				{"observe_transform.facets.ready_containers", int64(3)},
			},
		},
		{
			name: "Pod readiness gates",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEventWithReadinessGates.json",
				},
			),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.readinessGatesReady", int64(1)},
				{"observe_transform.facets.readinessGatesTotal", int64(2)},
			},
		},
		{
			name: "Pod conditions",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEvent.json",
				},
			),
			expectedResults: []queryWithResult{
				// Conditions must be a map with 5 elements
				{"observe_transform.facets.conditions | length(@)", float64(5)},
				{"observe_transform.facets.conditions.PodReadyToStartContainers", false},
				{"observe_transform.facets.conditions.Initialized", true},
				{"observe_transform.facets.conditions.Ready", false},
				{"observe_transform.facets.conditions.ContainersReady", false},
				{"observe_transform.facets.conditions.PodScheduled", true},
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			kep := newK8sEventsProcessor(zap.NewNop(), &Config{})
			logs, err := kep.processLogs(context.Background(), test.inLogs)
			require.NoError(t, err)
			// Since we don't do correlation among different logs, each testcase
			// "Logs" contains only one ResourceLog with one ScopeLog and a single LogRecord
			out := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes().AsRaw()
			for _, query := range test.expectedResults {
				queryJmes, err := jmespath.Compile(query.path)
				require.NoErrorf(t, err, "path %v is not a valid jmespath", query.path)
				res, err := queryJmes.Search(out)
				require.NoError(t, err, "error in extracting jmespath")
				require.Equal(t, query.expResult, res, "Processed log doesn't match the expected query")
			}
		})
	}
}

// Creates Logs with a single log.
func createResourceLogs(lwr logWithResource) plog.Logs {
	ld := plog.NewLogs()

	rl := ld.ResourceLogs().AppendEmpty()

	// Add resource level attributes
	//nolint:errcheck
	rl.Resource().Attributes().FromRaw(lwr.resourceAttributes)
	ls := rl.ScopeLogs().AppendEmpty().LogRecords()
	l := ls.AppendEmpty()
	// Add record level attributes
	//nolint:errcheck
	l.Attributes().FromRaw(lwr.recordAttributes)
	l.Attributes().PutStr("name", lwr.logName)
	// Set body & severity fields
	if lwr.body != "" {
		// Check that the body is a valid json
		_, err := json.Marshal(lwr.body)
		if err != nil {
			panic(err)
		}
		l.Body().SetStr(lwr.body)
	} else if lwr.testBodyFilepath != "" {
		file, err := os.ReadFile(lwr.testBodyFilepath)
		if err == nil {
			l.Body().SetStr(string(file))
		}
	}
	l.SetSeverityText(lwr.severityText)
	l.SetSeverityNumber(lwr.severityNumber)
	return ld
}
