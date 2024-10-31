package observek8sattributesprocessor

import "testing"

func TestPodActions(t *testing.T) {

	for _, test := range []k8sEventProcessorTest{
		{
			name: "noObserveTransformAttributes",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEvent.json",
				},
			),
			expectedResults: []queryWithResult{
				{
					path: "observe_transform.facets.status",
					expResult: "Terminating",
				},
			},
		},
		{ // Tests that we don't override/drop other facets computed in OTTL
			name: "existingObserveTransformAttributes",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEvent.json",
					recordAttributes: map[string]any{
						"observe_transform": map[string]interface{}{
							"facets": map[string]interface{}{
								"other_key": "test",
							},
						},
						"name": "existingObserveTransformAttributes",
					},
				},
			),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.status",
					expResult: "Terminating",
				},
				{
					path:      "observe_transform.facets.other_key",
					expResult: "test",
				},
			},
		},
		{
			name: "Pod container counts",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEvent.json",
				},
			),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.restarts",
					expResult: int64(5),
				},
				{
					path:      "observe_transform.facets.total_containers",
					expResult: int64(4),
				},
				{
					path:      "observe_transform.facets.ready_containers",
					expResult: int64(3),
				},
			},
		},
		{
			name: "Pod readiness gates",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEventWithReadinessGates.json",
				},
			),
			expectedResults: []queryWithResult{
				{
					path:      "observe_transform.facets.readinessGatesReady",
					expResult: int64(1),
				},
				{
					path:      "observe_transform.facets.readinessGatesTotal",
					expResult: int64(2),
				},
			},
		},
		{
			name: "Pod conditions",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podObjectEvent.json",
				},
			),
			expectedResults: []queryWithResult{
				// Conditions must be a map with 5 elements
				{
					path:      "observe_transform.facets.conditions | length(@)",
					expResult: float64(6),
				},
				{
					path:      "observe_transform.facets.conditions.PodReadyToStartContainers",
					expResult: "False",
				},
				{
					path:      "observe_transform.facets.conditions.Initialized",
					expResult: "True",
				},
				{
					path:      "observe_transform.facets.conditions.Ready",
					expResult: "False",
				},
				{
					path:      "observe_transform.facets.conditions.ContainersReady",
					expResult: "False",
				},
				{
					path:      "observe_transform.facets.conditions.PodScheduled",
					expResult: "True",
				},
				{
					path:      "observe_transform.facets.conditions.TestCondition",
					expResult: "Unknown",
				},
			},
		},
		{
			name: "Pod conditions",
			inLogs: createResourceLogs(
				logWithResource{
					testBodyFilepath: "./testdata/podTestEvent.json",
				},
			),
			expectedResults: []queryWithResult{
				// Conditions must be a map with 5 elements
				{
					path:      "observe_transform.facets.conditions | length(@)",
					expResult: float64(5),
				},
			},
		},
	} {
		runTest(t, test)
	}

}
