package observek8sattributesprocessor

import (
	"encoding/json"
	"errors"

	"strings"

	"go.opentelemetry.io/collector/pdata/plog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	NodeRolesAttributeKey = "roles"
	// labelNodeRolePrefix is a label prefix for node roles
	labelNodeRolePrefix = "node-role.kubernetes.io/"

	// nodeLabelRole specifies the role of a node
	nodeLabelRole = "kubernetes.io/role"
)

var NodeRolesAction = K8sEventProcessorAction{
	ComputeAttributes: getNodeRoles,
	// Reuse the function to filter events for nodes
	FilterFn: filterNodeEvents,
}

// Generates the Node "status" facet. Assumes that objLog is a log from a Node event.
func getNodeRoles(objLog plog.LogRecord) (attributes, error) {
	var node v1.Node
	err := json.Unmarshal([]byte(objLog.Body().AsString()), &node)
	if err != nil {
		return nil, errors.New("could not unmarshal Node")
	}
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

	ret := make([]any, 0, roles.Len())
	for _, role := range roles.List() {
		ret = append(ret, role)
	}
	return attributes{NodeRolesAttributeKey: ret}, nil
}
