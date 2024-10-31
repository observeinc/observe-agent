package observek8sattributesprocessor

import "testing"

func TestJobActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Running Job",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/jobRunningEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.status",
					expResult: "Running",
				},
			},
		},
		{
			name:   "Completed Job",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/jobCompletedEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.status",
					expResult: "Complete",
				},
			},
		},
		{
			name:   "Failed Job",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/jobCompletedEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.status",
					expResult: "Complete",
				},
			},
		},
		{
			name:   "Duration of completed job",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/jobCompletedEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.duration",
					expResult: "3m23s",
				},
			},
		},
	} {
		runTest(t, testCase)
	}
}
