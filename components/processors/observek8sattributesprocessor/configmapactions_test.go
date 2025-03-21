package observek8sattributesprocessor

import "testing"

func TestConfigMapActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "ConfigMap data",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/configMapEvent.json"),
			expectedResults: []queryWithResult{
				{path: "observe_transform.facets.data", expResult: int64(1)},
			},
		},
	} {
		runTest(t, testCase)
	}
}
