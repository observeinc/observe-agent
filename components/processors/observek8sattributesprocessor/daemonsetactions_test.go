package observek8sattributesprocessor

import "testing"

func TestDaemonSetActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Pretty print of a DaemonSet's selector",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/daemonSetEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.selector", "app.kubernetes.io/instance=observe-agent,app.kubernetes.io/name=daemonset-logs-metrics,component=agent-collector"},
			},
		},
	} {
		runTest(t, testCase, LogLocationAttributes)
	}
}
