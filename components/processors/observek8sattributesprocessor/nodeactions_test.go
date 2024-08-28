package observek8sattributesprocessor

import "testing"

func TestNodeActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name: "Node Status and Role",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventSimple.json",
				},
			),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Ready"},
				{"observe_transform.facets.roles | length(@)", float64(1)},
				{"observe_transform.facets.roles[0]", "control-plane"},
			},
		},
		{
			name: "Node Status and Multiple Roles",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/nodeObjectEventAlternativeRoleKey.json",
				},
			),
			expectedResults: []queryWithResult{
				{"observe_transform.facets.status", "Ready"},
				{"observe_transform.facets.roles | length(@)", float64(2)},
				{"observe_transform.facets.roles", []any{"anotherRole!", "control-plane"}},
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
				{"observe_transform.facets.nodePool", "test-node-group"},
			},
		},
	} {
		runTest(t, testCase)
	}

}
