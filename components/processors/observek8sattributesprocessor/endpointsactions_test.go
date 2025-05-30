package observek8sattributesprocessor

import "testing"

func TestEndpointsActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Endpoints",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/endpointsEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.endpoints",
					expResult: []any{"10.244.0.53:5432"},
				},
			},
		},
	} {
		runTest(t, testCase)
	}
}
