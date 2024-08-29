package observek8sattributesprocessor

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DaemonsetSelectorAttributeKey = "selector"
)

type DaemonSetSelectorAction struct{}

func NewDaemonsetSelectorAction() DaemonSetSelectorAction {
	return DaemonSetSelectorAction{}
}

// ---------------------------------- Daemonset "selector" ----------------------------------

// Generates the Daemonset "status" facet. Same logic as kubectl printer
// https://github.com/kubernetes/kubernetes/blob/0d3b859af81e6a5f869a7766c8d45afd1c600b04/pkg/printers/internalversion/printers.go#L1204
func (DaemonSetSelectorAction) ComputeAttributes(daemonset appsv1.DaemonSet) (attributes, error) {
	selecotString := metav1.FormatLabelSelector(daemonset.Spec.Selector)
	return attributes{DaemonsetSelectorAttributeKey: selecotString}, nil
}
