package observek8sattributesprocessor

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	ConfigMapDataAttributeKey = "data"
)

// ---------------------------------- ConfigMap "endpoints" ----------------------------------

type ConfigMapDataAction struct{}

func NewConfigMapDataAction() ConfigMapDataAction {
	return ConfigMapDataAction{}
}

// Generates the ConfigMap "endpoints" facet, which is a list of all individual endpoints, encoded as strings
func (ConfigMapDataAction) ComputeAttributes(configMap corev1.ConfigMap) (attributes, error) {
	return attributes{ConfigMapDataAttributeKey: len(configMap.Data) + len(configMap.BinaryData)}, nil
}
