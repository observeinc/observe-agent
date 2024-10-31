package observek8sattributesprocessor

import (
	"fmt"
	"testing"
)

func TestCronJobActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Active CronJob jobs",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/cronJobEvent.json"),
			expectedResults: []queryWithResult{
				{path: fmt.Sprintf("observe_transform.facets.%s", CronJobActiveKey), expResult: int64(1)},
			},
		},
		{
			name:   "Idle CronJob jobs",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/cronJobEventNotActive.json"),
			expectedResults: []queryWithResult{
				{path: fmt.Sprintf("observe_transform.facets.%s", CronJobActiveKey), expResult: int64(0)},
			},
		},
	} {
		runTest(t, testCase)
	}
}
