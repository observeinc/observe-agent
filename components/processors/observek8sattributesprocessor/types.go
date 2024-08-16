package observek8sattributesprocessor

import (
	"errors"

	v1 "k8s.io/api/core/v1"
)

type K8sEvent struct {
	Kind       string `json:"kind,omitempty"`
	ApiVersion string `json:"apiVersion,omitempty"`
}

type attributes map[string]any

// Action that processes a K8S object and computes custom attributes for it
type K8sEventProcessorAction interface {
	// Computes attributes for a k8s entity.
	// Since entities don't have a common interface, we use any here and each
	// instance of this struct should check that the input object is of the
	// required type
	ComputeAttributes(any) (attributes, error)
}

// The following type aliases allow for safely adding new actions.  They prevent
// developers from adding actions that compute attributes for different input
// object types together.

// A nodeAction is any K8sEventProcessorAction that is able to compute
// attributes for events of type Node
// type nodeAction K8sEventProcessorAction

func getNode(obj any) (v1.Node, error) {
	node, ok := obj.(v1.Node)
	if !ok {
		return node, errors.New("cannot compute Node Roles. Input is not a Node object")
	}
	return node, nil
}

// A NodeAction is any K8sEventProcessorAction that is able to compute
// attributes for events of type Pod
// type podAction K8sEventProcessorAction

func getPod(obj any) (v1.Pod, error) {
	pod, ok := obj.(v1.Pod)
	if !ok {
		return pod, errors.New("cannot compute Node Roles. Input is not a Pod object")
	}
	return pod, nil
}
