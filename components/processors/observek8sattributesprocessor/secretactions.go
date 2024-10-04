package observek8sattributesprocessor

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	RedactedSecretValue = "REDACTED"
)

type SecretRedactorBodyAction struct{}

func NewSecretRedactorBodyAction() SecretRedactorBodyAction {
	return SecretRedactorBodyAction{}
}

// ---------------------------------- Secret "data" values' redaction ----------------------------------

// Redacts secrets' values
func (SecretRedactorBodyAction) Modify(secret *corev1.Secret) error {
	for key := range secret.Data {
		secret.Data[key] = []byte(RedactedSecretValue)
	}
	return nil
}
