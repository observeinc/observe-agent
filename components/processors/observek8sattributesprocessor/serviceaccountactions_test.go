package observek8sattributesprocessor

import "testing"

func TestServiceAccountActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Service account secrets",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceAccountEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.secretsNames", []any{"example-another-secret", "example-serviceaccount-token-abcdef"}},
			},
		},
	} {
		runTest(t, testCase)
	}
}
