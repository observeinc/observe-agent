package observek8sattributesprocessor

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ServiceAccountSecretsNamesAttributeKey     = "secretsNames"
	ServiceAccountSecretsAttributeKey          = "secrets"
	ServiceAccountImagePullSecretsAttributeKey = "imagePullSecrets"
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

// ---------------------------------- ServiceAccount "secrets" ----------------------------------

type ServiceAccountSecretsAction struct{}

func NewServiceAccountSecretsAction() ServiceAccountSecretsAction {
	return ServiceAccountSecretsAction{}
}

// Generates the ServiceAccount "secrets" facet.
func (ServiceAccountSecretsAction) ComputeAttributes(serviceAccount corev1.ServiceAccount) (attributes, error) {
	return attributes{ServiceAccountSecretsAttributeKey: len(serviceAccount.Secrets)}, nil
}

// ---------------------------------- ServiceAccount "imagePullSecrets" ----------------------------------

type ServiceAccountImagePullSecretsAction struct{}

func NewServiceAccountImagePullSecretsAction() ServiceAccountImagePullSecretsAction {
	return ServiceAccountImagePullSecretsAction{}
}

// Generates the ServiceAccount "ImagePullSecrets" facet.
func (ServiceAccountImagePullSecretsAction) ComputeAttributes(serviceAccount corev1.ServiceAccount) (attributes, error) {
	return attributes{ServiceAccountImagePullSecretsAttributeKey: len(serviceAccount.ImagePullSecrets)}, nil
}
