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

// ---------------------------------- DaemonSet "selector" ----------------------------------

// Generates the Daemonset "selector" facet.
func (DaemonSetSelectorAction) ComputeAttributes(daemonset appsv1.DaemonSet) (attributes, error) {
	selectorString := metav1.FormatLabelSelector(daemonset.Spec.Selector)
	return attributes{DaemonsetSelectorAttributeKey: selectorString}, nil
}
