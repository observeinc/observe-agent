package observek8sattributesprocessor

import "testing"

func TestServiceAccountActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Service account secrets' names",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceAccountEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.secretsNames", []any{"example-another-secret", "example-serviceaccount-token-abcdef"}},
			},
		},
		{
			name:   "Service account secrets",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceAccountEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.secrets", int64(2)},
			},
		},
		{
			name:   "Service account imagePull secrets",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceAccountEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.imagePullSecrets", int64(0)},
			},
		},
	} {
		runTest(t, testCase, LogLocationAttributes)
	}
}
