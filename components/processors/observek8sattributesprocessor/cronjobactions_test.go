package observek8sattributesprocessor

import "testing"

func TestCronJobActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Active CronJob",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/cronJobEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.active", int64(1)},
			},
		},
		{
			name:   "Idle CronJob",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/cronJobEventNotActive.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.active", int64(0)},
			},
		},
	} {
		runTest(t, testCase, LogLocationAttributes)
	}
}
