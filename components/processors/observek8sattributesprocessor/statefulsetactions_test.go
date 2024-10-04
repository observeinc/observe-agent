package observek8sattributesprocessor

import "testing"

func TestStatefulSetActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Pretty print of a StatefulSet's selector",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/statefulSetEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.selector", "app=redis-ephemeral"},
			},
		},
	} {
		runTest(t, testCase, LogLocationAttributes)
	}
}
