package observek8sattributesprocessor

import "testing"

func TestIngressActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Ingress rules with host",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/ingressEvent.json"),
			expectedResults: []queryWithResult{
				{
					path: "observe_transform.facets.rules",
					expResult: []any{
						map[string]any{
							"host": "prometheus.observe-eng.com",
							"httpRules": []any{
								map[string]any{
									"backend": map[string]any{
										"service": map[string]any{
											"name": "prometheus",
											"port": "prometheus",
										},
									},
									"path": "/",
								}}}},
				},
			},
		},
		{
			name:   "Ingress rules without host",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/ingressEvent2.json"),
			expectedResults: []queryWithResult{
				{
					path: "observe_transform.facets.rules",
					expResult: []any{
						map[string]any{
							"host": "*",
							"httpRules": []any{
								map[string]any{
									"backend": map[string]any{
										"service": map[string]any{
											"name": "test",
											"port": int64(80),
										},
									},
									"path": "/testpath",
								}},
						},
						// Rule with resource backend
						map[string]any{
							"host": "test.com",
							"httpRules": []any{
								map[string]any{
									"backend": map[string]any{
										"resource": "testResource",
									},
									"path": "/testpath2",
								}},
						},
					}},
			},
		},
		{
			name:   "Load Balancer",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/ingressEvent.json"),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.loadBalancer",
					expResult: "someUniqueElbIdentifier.elb.us-west-2.amazonaws.com",
				},
			},
		},
	} {
		runTest(t, testCase)
	}
}
