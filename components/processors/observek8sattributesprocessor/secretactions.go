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

func redactSecretKeys(secret *corev1.Secret) {
	for key := range secret.Data {
		secret.Data[key] = []byte(RedactedSecretValue)
	}
	for key := range secret.StringData {
		secret.StringData[key] = RedactedSecretValue
	}
}

// Redacts secrets' values
func (SecretRedactorBodyAction) Modify(secret *corev1.Secret) error {
	redactSecretKeys(secret)

	annotations := secret.GetAnnotations()
	delete(annotations, corev1.LastAppliedConfigAnnotation)
	secret.SetAnnotations(annotations)
	return nil
}
