package observek8sattributesprocessor

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	StatefulsetSelectorAttributeKey = "selector"
)

// ---------------------------------- StatefulSet "selector" ----------------------------------

type StatefulSetSelectorAction struct{}

func NewStatefulsetSelectorAction() StatefulSetSelectorAction {
	return StatefulSetSelectorAction{}
}

// Generates the Statefulset "selector" facet.
func (StatefulSetSelectorAction) ComputeAttributes(statefulSet appsv1.StatefulSet) (attributes, error) {
	selecotString := metav1.FormatLabelSelector(statefulSet.Spec.Selector)
	return attributes{StatefulsetSelectorAttributeKey: selecotString}, nil
}
