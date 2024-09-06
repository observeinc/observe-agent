package observek8sattributesprocessor

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	ConfigMapDataAttributeKey = "data"
)

// ---------------------------------- ConfigMap "data" ----------------------------------

type ConfigMapDataAction struct{}

func NewConfigMapDataAction() ConfigMapDataAction {
	return ConfigMapDataAction{}
}

// Generates the ConfigMap "data" facet, calculated as the total number of entries in data and binaryData
func (ConfigMapDataAction) ComputeAttributes(configMap corev1.ConfigMap) (attributes, error) {
	return attributes{ConfigMapDataAttributeKey: len(configMap.Data) + len(configMap.BinaryData)}, nil
}
