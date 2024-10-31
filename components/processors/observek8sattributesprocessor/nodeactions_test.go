package observek8sattributesprocessor

import "testing"

func TestNodeActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name: "Node is Ready",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventSimple.json",
				},
			),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.status",
					expResult: "Ready",
				},
			},
		},
		{
			name: "Node is NOT Ready",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventNotReady.json",
				},
			),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.status",
					expResult: "NotReady",
				},
			},
		},
		{
			name: "Node Single Role",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventSimple.json",
				},
			),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.roles | length(@)",
					expResult: float64(1)},
				{
					path:      "observe_transform.facets.roles[0]",
					expResult: "control-plane",
				},
			},
		},
		{
			name: "Node Multiple Roles",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventAlternativeRoleKey.json",
				},
			),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.roles | length(@)",
					expResult: float64(2)},
				{
					path:      "observe_transform.facets.roles",
					expResult: []any{"anotherRole!", "control-plane"},
				},
			},
		},
		{
			name: "Node Pool",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventSimple.json",
				},
			),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.nodePool",
					expResult: "test-node-group",
				},
			},
		},
	} {
		runTest(t, testCase)
	}

}
