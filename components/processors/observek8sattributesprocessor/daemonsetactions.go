package observek8sattributesprocessor

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DaemonSetSelectorAttributeKey = "selector"
)

type DaemonSetSelectorAction struct{}

func NewDaemonSetSelectorAction() DaemonSetSelectorAction {
	return DaemonSetSelectorAction{}
}

// ---------------------------------- DaemonSet "selector" ----------------------------------

// Generates the DaemonSet "selector" facet.
func (DaemonSetSelectorAction) ComputeAttributes(daemonset appsv1.DaemonSet) (attributes, error) {
	selecotString := metav1.FormatLabelSelector(daemonset.Spec.Selector)
	return attributes{DaemonSetSelectorAttributeKey: selecotString}, nil
}
