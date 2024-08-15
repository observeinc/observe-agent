package observek8sattributesprocessor

import (
	"encoding/json"
	"errors"

	"go.opentelemetry.io/collector/pdata/plog"
	v1 "k8s.io/api/core/v1"
)

const (
	// This action will be ignored and not written in any of the facets, since
	// we return map[string]any
	PodReadinessGatesReadyAttributeKey = "readinessGatesReady"
	PodReadinessGatesTotalAttributeKey = "readinessGatesTotal"
)

// This action computes various facets for Pod by aggregating "status" values
// across all containers of a Pod.
//
// We compute more facets into a single action to avoid iterating over the
// same slice multiple times in different actions.
var PodReadinessAction = K8sEventProcessorAction{
	ComputeAttributes: getPodReadiness,
	FilterFn:          filterPodEvents,
}

func getPodReadiness(objLog plog.LogRecord) (attributes, error) {
	var pod v1.Pod
	err := json.Unmarshal([]byte(objLog.Body().AsString()), &pod)
	if err != nil {
		return nil, errors.New("could not unmarshal Pod")
	}
	readinessGatesReady := 0

	if len(pod.Spec.ReadinessGates) > 0 {
		for _, readinessGate := range pod.Spec.ReadinessGates {
			conditionType := readinessGate.ConditionType
			for _, condition := range pod.Status.Conditions {
				if condition.Type == conditionType {
					if condition.Status == v1.ConditionTrue {
						readinessGatesReady++
					}
					break
				}
			}
		}
	}

	return attributes{
		PodReadinessGatesTotalAttributeKey: len(pod.Spec.ReadinessGates),
		PodReadinessGatesReadyAttributeKey: readinessGatesReady,
	}, nil
}
