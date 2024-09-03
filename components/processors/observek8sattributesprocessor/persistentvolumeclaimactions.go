package observek8sattributesprocessor

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PersistentVolumeClaimSelectorAttributeKey = "selector"
)

type PersistentVolumeClaimSelectorAction struct{}

func NewPersistentVolumeClaimSelectorAction() PersistentVolumeClaimSelectorAction {
	return PersistentVolumeClaimSelectorAction{}
}

// Generates the PersistentVolumeClaim "selector" facet.
func (PersistentVolumeClaimSelectorAction) ComputeAttributes(pvc corev1.PersistentVolumeClaim) (attributes, error) {
	selecotString := metav1.FormatLabelSelector(pvc.Spec.Selector)
	return attributes{PersistentVolumeClaimSelectorAttributeKey: selecotString}, nil
}
