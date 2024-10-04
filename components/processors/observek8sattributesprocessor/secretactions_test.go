package observek8sattributesprocessor

import (
	"encoding/base64"
	"fmt"
	"testing"
)

func TestSecretBodyActions(t *testing.T) {
	for _, testCase := range []k8sEventProcessorTest{
		{
			name:   "Redact secrets' values",
			inLogs: resourceLogsFromSingleJsonEvent("./testdata/secretEvent.json"),
			expectedResults: []queryWithResult{
				// This checks that there are no values in "data" that are not "REDACTED"
				{fmt.Sprintf("length(values(data)[?@ != '%s'])", base64.StdEncoding.EncodeToString([]byte(RedactedSecretValue))), float64(0)},
			},
		},
	} {
		runTest(t, testCase, LogLocationBody)
	}
}
