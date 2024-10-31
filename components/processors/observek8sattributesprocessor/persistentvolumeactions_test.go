package observek8sattributesprocessor

import "testing"

func TestPersistentVolumeActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Extract PersistentVolume type (AWSElasticBlockStore)",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/persistentVolumeAWSElasticBlockStoreEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.volumeType",
					expResult: "AWSElasticBlockStore",
				},
			},
		},
		{
			name:   "Extract PersistentVolume type (HostPath)",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/persistentVolumeHostPathEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.volumeType",
					expResult: "HostPath",
				},
			},
		},
	} {
		runTest(t, testCase)
	}
}
