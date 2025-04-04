package observek8sattributesprocessor

import (
	"encoding/base64"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestSecretBodyActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Redact secrets' values",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/secretEvent.json"),
			expectedResults: []queryWithResult{
				// This checks that there are no values in "data" that are not "REDACTED"
				{
					location:  LogLocationBody,
					path:      fmt.Sprintf("length(values(data)[?@ != '%s'])", base64.StdEncoding.EncodeToString([]byte(RedactedSecretValue))),
					expResult: float64(0),
				},
			},
		},
		{
			name:   "Redact secrets' last configuration values",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/secretEventPrevConfig.json"),
			expectedResults: []queryWithResult{
				{
					location:  LogLocationBody,
					path:      fmt.Sprintf("metadata.annotations.\"%s\"", corev1.LastAppliedConfigAnnotation),
					expResult: nil,
				},
			},
		},
	} {
		runTest(t, testCase)
	}
}
