package observek8sattributesprocessor

import "testing"

func TestIngressActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Ingress rules",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/ingressEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.rules", "prometheus.observe-eng.com"},
			},
		},
		{
			name:   "Ingress rules",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/ingressEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.loadBalancer", "someUniqueElbIdentifier.elb.us-west-2.amazonaws.com"},
			},
		},
	} {
		runTest(t, testCase)
	}
}
