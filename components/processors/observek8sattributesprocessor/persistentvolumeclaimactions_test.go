package observek8sattributesprocessor

import "testing"

func TestPersistentVolumeClaimActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Pretty print of a PersistentVolumeClaim's selector",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/persistentVolumeClaimEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.selector", "environment in (production,staging),storage-tier=high-performance"},
			},
		},
	} {
		runTest(t, testCase, LogLocationAttributes)
	}
}
