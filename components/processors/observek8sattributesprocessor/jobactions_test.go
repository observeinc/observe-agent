package observek8sattributesprocessor

import "testing"

func TestJobActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Running Job",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/jobRunningEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Running"},
			},
		},
		{
			name:   "Completed Job",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/jobCompletedEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Complete"},
			},
		},
		{
			name:   "Failed Job",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/jobCompletedEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Complete"},
			},
		},
	} {
		runTest(t, testCase)
	}
}
