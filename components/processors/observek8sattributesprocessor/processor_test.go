package observek8sattributesprocessor

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/jmespath/go-jmespath"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type logWithResource struct {
	logName            string
	resourceAttributes map[string]any
	// These attributes simulate the enrichments performed on top of the raw
	// event by the pipelines in the agent config (generated from the template
	// in the helm repo)
	recordAttributes map[string]any
	severityText     string
	body             string
	testBodyFilepath string
	severityNumber   plog.SeverityNumber
}

// models a jmespath query against the processed results
type queryWithResult struct {
	path      string
	expResult any
}

// Functions that check the body of a logRecord post processor actions
// These "custom" check functions are a powerful way to introspect the body
// of a log record (as a raw JSON string).
// Implementations of these functions most likely start with unmarshalling
// the body string to an API object of choice
type checkBodyFunc func(t *testing.T, body string)

// Functions that check the attributes of a logRecord post processor actions
// These custom actions take as input the key-value attributes as a
// map[string]any
type checkAttributesFunc func(t *testing.T, attributes map[string]any) //nolint:unused

type k8sEventProcessorTest struct {
	name            string
	inLogs          plog.Logs
	expectedResults []queryWithResult
	// Actions that are only ran when testing the body of a resulting logRecord
	checkBodyFunctions []checkBodyFunc
	// Actions that are only ran when testing the attributes a resulting
	// logRecord
	checkAttributesFunctions []checkAttributesFunc //nolint:unused
}

// LogLocation is the part of the log where to check for matches. At the moment,
// it can only be body or attributes, we might add resource_attributes in the
// future.
type LogLocation string

const (
	LogLocationBody       LogLocation = "body"
	LogLocationAttributes LogLocation = "attributes"
)

func runTest(t *testing.T, test k8sEventProcessorTest, location LogLocation) {
	t.Run(test.name, func(t *testing.T) {
		kep := newK8sEventsProcessor(zap.NewNop(), &Config{})
		logs, err := kep.processLogs(context.Background(), test.inLogs)
		require.NoError(t, err)
		// Since we don't do correlation among different logs, each testcase
		// "Logs" contains only one ResourceLog with one ScopeLog and a single LogRecord
		var out map[string]any
		logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		switch location {
		case LogLocationBody:
			// The body is JSON string and therefore must be unmarshalled into
			// map[string]any to be able to query it with jmespath.
			body := logRecord.Body().AsString()
			json.Unmarshal([]byte(body), &out)
			for _, fn := range test.checkBodyFunctions {
				fn(t, body)
			}
		case LogLocationAttributes:
			out = logRecord.Attributes().AsRaw()
			for _, fn := range test.checkAttributesFunctions {
				fn(t, out)
			}
		}
		for _, query := range test.expectedResults {
			queryJmes, err := jmespath.Compile(query.path)
			require.NoErrorf(t, err, "path %v is not a valid jmespath", query.path)
			res, err := queryJmes.Search(out)
			require.NoError(t, err, "error in extracting jmespath")
			require.Equal(t, query.expResult, res, "Processed log doesn't match the expected query")
		}
	})
}

// TODO [eg] Understand if we should refactor this to simplify it. At the end of
// the day we are only testing computing attributes from a single event, which
// is always coming from a JSON file under testdata/.

func resourceLogsFromSingleJsonEvent(path string) plog.Logs {
	return createResourceLogs(logWithResource{testBodyFilepath: path})
}

// Creates Logs with a single log.
func createResourceLogs(lwr logWithResource) plog.Logs {
	ld := plog.NewLogs()

	rl := ld.ResourceLogs().AppendEmpty()

	// Add resource level attributes
	//nolint:errcheck
	rl.Resource().Attributes().FromRaw(lwr.resourceAttributes)
	ls := rl.ScopeLogs().AppendEmpty().LogRecords()
	l := ls.AppendEmpty()
	// Add record level attributes
	//nolint:errcheck
	l.Attributes().FromRaw(lwr.recordAttributes)
	l.Attributes().PutStr("name", lwr.logName)
	// Set body & severity fields
	if lwr.body != "" {
		// Check that the body is a valid json
		_, err := json.Marshal(lwr.body)
		if err != nil {
			panic(err)
		}
		l.Body().SetStr(lwr.body)
	} else if lwr.testBodyFilepath != "" {
		file, err := os.ReadFile(lwr.testBodyFilepath)
		if err == nil {
			l.Body().SetStr(string(file))
		}
	}
	l.SetSeverityText(lwr.severityText)
	l.SetSeverityNumber(lwr.severityNumber)
	return ld
}
