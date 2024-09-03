package observek8sattributesprocessor

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ServiceAccountSecretsNamesAttributeKey = "secretsNames"
)

// ---------------------------------- ServiceAccount "secretsNames" ----------------------------------
type ServiceAccountSecretsNamesAction struct{}

func NewServiceAccountSecretsNamesAction() ServiceAccountSecretsNamesAction {
	return ServiceAccountSecretsNamesAction{}
}

// Generates the ServiceAccount "secretsNames" facet.
func (ServiceAccountSecretsNamesAction) ComputeAttributes(serviceAccount corev1.ServiceAccount) (attributes, error) {
	result := sets.NewString()
	for _, secret := range serviceAccount.Secrets {
		result.Insert(secret.Name)
	}

	return attributes{ServiceAccountSecretsNamesAttributeKey: result.List()}, nil
}
