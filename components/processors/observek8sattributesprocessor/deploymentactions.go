package observek8sattributesprocessor

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeploymentSelectorAttributeKey = "selector"
)

type DeploymentSelectorAction struct{}

func NewDeploymentSelectorAction() DeploymentSelectorAction {
	return DeploymentSelectorAction{}
}

// ---------------------------------- Deployment "selector" ----------------------------------

// Generates the Deployment "selector" facet.
func (DeploymentSelectorAction) ComputeAttributes(deployment appsv1.Deployment) (attributes, error) {
	selecotString := metav1.FormatLabelSelector(deployment.Spec.Selector)
	return attributes{DeploymentSelectorAttributeKey: selecotString}, nil
}
