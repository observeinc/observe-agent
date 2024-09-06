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
				{"observe_transform.facets.status", "Terminating"},
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
				{"observe_transform.facets.status", "Terminating"},
				{"observe_transform.facets.other_key", "test"},
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
				{"observe_transform.facets.restarts", int64(5)},
				{"observe_transform.facets.total_containers", int64(4)},
				{"observe_transform.facets.ready_containers", int64(3)},
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
				{"observe_transform.facets.readinessGatesReady", int64(1)},
				{"observe_transform.facets.readinessGatesTotal", int64(2)},
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
				{"observe_transform.facets.conditions | length(@)", float64(6)},
				{"observe_transform.facets.conditions.PodReadyToStartContainers", "False"},
				{"observe_transform.facets.conditions.Initialized", "True"},
				{"observe_transform.facets.conditions.Ready", "False"},
				{"observe_transform.facets.conditions.ContainersReady", "False"},
				{"observe_transform.facets.conditions.PodScheduled", "True"},
				{"observe_transform.facets.conditions.TestCondition", "Unknown"},
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
				{"observe_transform.facets.conditions | length(@)", float64(5)},
			},
		},
	} {
		runTest(t, test)
	}

}
