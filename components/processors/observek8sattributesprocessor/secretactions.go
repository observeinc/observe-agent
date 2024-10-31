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

// Obfuscates secrets' values in place
func redactSecretKeys(secret *corev1.Secret) {
	// These will be encoded in base64 when serialized.  While it would be nice,
	// don't expect to see "REDACTED" in plaintext in the processed secrets'
	// "data".
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
