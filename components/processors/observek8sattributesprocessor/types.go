package observek8sattributesprocessor

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

type K8sEvent struct {
	Kind       string `json:"kind,omitempty"`
	ApiVersion string `json:"apiVersion,omitempty"`
}

type attributes map[string]any

// Action that processes a K8S object and computes custom attributes for it
type K8sEventProcessorAction interface {
	// Computes attributes for a k8s entity.  Since entities like Pod, Node,
	// etc. don't have a common interface, the argument of ComputeAttributes is
	// of type any types that implement this method should check that the arg is
	// of the right type before proceeding.
	// Check the utility functions below for more info
	ComputeAttributes(any) (attributes, error)
}

// Type asserts obj to v1.Node, returning an error if the underlying concrete
// value is not of type Node
func getNode(obj any) (v1.Node, error) {
	node, ok := obj.(v1.Node)
	if !ok {
		return node, fmt.Errorf("cannot convert %v to Node", obj)
	}
	return node, nil
}

// Type asserts obj to v1.Pod, returning an error if the underlying concrete
// value is not of type Pod
func getPod(obj any) (v1.Pod, error) {
	pod, ok := obj.(v1.Pod)
	if !ok {
		return pod, fmt.Errorf("cannot convert %v to Pod", obj)
	}
	return pod, nil
}
