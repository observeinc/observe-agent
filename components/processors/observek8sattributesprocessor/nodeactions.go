package observek8sattributesprocessor

import (
	"strings"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	NodeStatusAttributeKey = "status"

	NodePoolAttributeKey = "nodePool"

	NodeRolesAttributeKey = "roles"
	// labelNodeRolePrefix is a label prefix for node roles
	labelNodeRolePrefix = "node-role.kubernetes.io/"

	// nodeLabelRole specifies the role of a node
	nodeLabelRole = "kubernetes.io/role"
)

type NodeStatusAction struct{}

func NewNodeStatusAction() NodeStatusAction {
	return NodeStatusAction{}
}

// ---------------------------------- Node "status" ----------------------------------

// Generates the Node "status" facet. Assumes that objLog is a log from a Node event.
func (NodeStatusAction) ComputeAttributes(node v1.Node) (attributes, error) {
	// based on https://github.com/kubernetes/kubernetes/blob/dbc2b0a5c7acc349ea71a14e49913661eaf708d2/pkg/printers/internalversion/printers.go#L1835
	// Although with a simplified logic that is faster to compute and uses less memory
	var status string
	// For now, we only care about "Ready"/"Not Ready", that's why we simplify the logic
	for _, condition := range node.Status.Conditions {
		if condition.Type != v1.NodeReady {
			continue
		}
		status = string(condition.Type)
		if condition.Status == v1.ConditionFalse {
			status = "Not" + status
		}
	}
	// If there's no Ready condition in the status, use Unknown
	if status == "" {
		status = "Unknown"
	}
	if node.Spec.Unschedulable {
		status += ", SchedulingDisabled"
	}

	return attributes{NodeStatusAttributeKey: status}, nil
}

// ---------------------------------- Node "pool" ----------------------------------

type NodePoolAction struct{}

func NewNodePoolAction() NodePoolAction {
	return NodePoolAction{}
}

var providerNodepoolLabels = map[string]struct{}{
	"eks.amazonaws.com/nodegroup":        {}, // AWS
	"cloud.google.com/gke-nodepool":      {}, // GCP
	"kubernetes.azure.com/agentpool":     {}, // "AKS"
	"doks.digitalocean.com/node-pool-id": {}, // "DOKS"
}

// Discover the Node "pool" facet. This faceit is not provided natively by
// Kubernetes, so it will be present only when using a managed
// deployment/service provided by either of the vendors listed above.
func (NodePoolAction) ComputeAttributes(node v1.Node) (attributes, error) {
	pool := "none"
	for label, value := range node.Labels {
		if _, found := providerNodepoolLabels[label]; found {
			pool = value
			break
		}
	}
	return attributes{NodePoolAttributeKey: pool}, nil
}

// ---------------------------------- Node "pool" ----------------------------------

type NodeRolesAction struct{}

func NewNodeRolesAction() NodeRolesAction {
	return NodeRolesAction{}
}

// Generates the Node "status" facet. Assumes that objLog is a log from a Node event.
func (NodeRolesAction) ComputeAttributes(node v1.Node) (attributes, error) {
	// based on https://github.com/kubernetes/kubernetes/blob/dbc2b0a5c7acc349ea71a14e49913661eaf708d2/pkg/printers/internalversion/printers.go#L183https://github.com/kubernetes/kubernetes/blob/1e12d92a5179dbfeb455c79dbf9120c8536e5f9c/pkg/printers/internalversion/printers.go#L14875
	roles := sets.NewString()
	for k, v := range node.Labels {
		switch {
		// The role could be in the key and not in the value
		case strings.HasPrefix(k, labelNodeRolePrefix):
			if role := strings.TrimPrefix(k, labelNodeRolePrefix); len(role) > 0 {
				roles.Insert(role)
			}
		case k == nodeLabelRole && v != "":
			roles.Insert(v)
		}
	}

	return attributes{NodeRolesAttributeKey: roles.List()}, nil
}
