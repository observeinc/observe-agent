package observek8sattributesprocessor

import "testing"

func TestServiceActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Pretty print of a Service selector",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceClusterIPEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.selector", "app=redis-ephemeral"},
			},
		},
		{
			name:   "Service LB Ingress of non-LB service",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceClusterIPEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.loadBalancerIngress", nil},
			},
		},
		{
			name:   "Service LB Ingress of initializing LB service",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceLoadBalancerPendingEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.loadBalancerIngress", "<pending>"},
			},
		},
		{
			name:   "Service LB Ingress of working LB service",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceLoadBalancerIngressEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.loadBalancerIngress", "someLoadBalancerIdentifier.elb.us-west-2.amazonaws.com"},
			},
		},
		{
			name:   "Service ports",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceClusterIPEvent.json"),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.ports", "6379/TCP"},
			},
		},
	} {
		runTest(t, testCase, LogLocationAttributes)
	}
}
