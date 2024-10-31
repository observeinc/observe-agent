package observek8sattributesprocessor

import "testing"

func TestServiceActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Pretty print of a Service selector",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceClusterIPEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.selector",
					expResult: "app=redis-ephemeral",
				},
			},
		},
		{
			name:   "LB Ingress (working Service)",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceLoadBalancer.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.loadBalancerIngress",
					expResult: "someLoadBalancerIngressIdentifier.elb.us-west-2.amazonaws.com",
				},
			},
		},
		{
			name:   "Service ports",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceClusterIPEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.ports",
					expResult: "6379/TCP",
				},
			},
		},
		{
			name:   "External IPs",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceLoadBalancer.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.externalIPs",
					expResult: []interface{}{"someLoadBalancerIngressIdentifier.elb.us-west-2.amazonaws.com"},
				},
			},
		},
		{
			name:   "Pending LB",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceLoadBalancerPendingEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.externalIPs",
					expResult: "Pending",
				},
			},
		},
		{
			name:   "Unknown Service type",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceUnknown.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.externalIPs",
					expResult: "Unknown",
				},
			},
		},
		{
			name:   "No External IPs",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/serviceClusterIPEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.externalIPs",
					expResult: "None",
				},
			},
		},
	} {
		runTest(t, testCase)
	}
}
