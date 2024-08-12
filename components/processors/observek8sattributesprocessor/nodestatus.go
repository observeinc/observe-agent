package observek8sattributesprocessor

import (
	"encoding/json"

	"go.opentelemetry.io/collector/pdata/plog"
	apiv1 "k8s.io/api/core/v1"
)

const (
	NodeStatusAttributeKey = "status"
)

var NodeStatusAction = K8sEventProcessorAction{
	Key:      NodeStatusAttributeKey,
	ValueFn:  getNodeStatus,
	FilterFn: filterNodeEvents,
}

func filterNodeEvents(event K8sEvent) bool {
	return event.Kind == "Node"
}

// Generates the Node "status" facet. Assumes that objLog is a log from a Node event.
func getNodeStatus(objLog plog.LogRecord) any {
	var n apiv1.Node
	err := json.Unmarshal([]byte(objLog.Body().AsString()), &n)
	if err != nil {
		return "Error while computing status"
	}
	// based on https://github.com/kubernetes/kubernetes/blob/dbc2b0a5c7acc349ea71a14e49913661eaf708d2/pkg/printers/internalversion/printers.go#L1835
	// Although with a simplified logic that is faster to compute and uses less memory
	var status string
	// For now, we only care about "Ready"/"Not Ready", that's why we simplify the logic
	for _, condition := range n.Status.Conditions {
		if condition.Type != apiv1.NodeReady {
			continue
		}
		status = string(condition.Type)
		if condition.Status == apiv1.ConditionFalse {
			status = "Not" + status
		}
	}
	// If there's no Ready condition in the status, use Unknown
	if status == "" {
		status = "Unknown"
	}
	if n.Spec.Unschedulable {
		status += ", SchedulingDisabled"
	}

	return status
}
