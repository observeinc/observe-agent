package observek8sattributesprocessor

import "testing"

func TestDeploymentActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Pretty print of a DaemonSet's selector",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/deploymentEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.selector", "app.kubernetes.io/instance=observe-agent,app.kubernetes.io/name=deployment-cluster-events,component=standalone-collector"},
			},
		},
	} {
		runTest(t, testCase, LogLocationAttributes)
	}
}
