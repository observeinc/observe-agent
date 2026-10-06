package config

import (
	"bufio"
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/otelcol"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const (
	envNameKey   = "deployment.environment.name"
	envLegacyKey = "deployment.environment"
	otlpHTTPURL  = "http://127.0.0.1:24318"
)

// envAttrs holds the values of deployment.environment.name and deployment.environment
// on a resource; "<unset>" marks a missing key.
type envAttrs struct {
	Name   string
	Legacy string
}

// Each app resource is sent with a different combination of environment keys. The agent is
// configured with resource_attributes deployment.environment.name=prod, which must only be
// used as a fallback for resources that do not declare an environment themselves.
var appResources = []struct {
	service string
	attrs   map[string]string
}{
	{service: "no-env"},
	{service: "legacy-env", attrs: map[string]string{envLegacyKey: "staging"}},
	{service: "new-env", attrs: map[string]string{envNameKey: "dev"}},
}

func Test_DeploymentEnvironmentPrecedence(t *testing.T) {
	outDir := t.TempDir()
	t.Setenv("DEPLOYMENT_ENV_TEST_DIR", outDir)
	setupConfig(t, snapshotTest{
		agentConfigPath: "test/deployment-environment-agent-config.yaml",
		otelConfigPath:  "test/deployment-environment-otel-config.yaml",
		packageType:     Linux,
	})
	// setupConfig points file_storage at the packaged path, which does not exist in CI.
	t.Setenv("FILESTORAGE_PATH", t.TempDir())
	startCollector(t)

	postOTLP(t, otlpHTTPURL, "/v1/traces", buildTraces(t))
	postOTLP(t, otlpHTTPURL, "/v1/logs", buildLogs(t))
	postOTLP(t, otlpHTTPURL, "/v1/metrics", buildMetrics(t))

	want := map[string]envAttrs{
		"no-env":     {Name: "prod", Legacy: "prod"},
		"legacy-env": {Name: "staging", Legacy: "staging"},
		"new-env":    {Name: "dev", Legacy: "dev"},
	}

	t.Run("forwarded traces", func(t *testing.T) {
		assert.Equal(t, want, waitForResources(t, filepath.Join(outDir, "traces.jsonl"), tracesResources))
	})
	t.Run("forwarded logs", func(t *testing.T) {
		assert.Equal(t, want, waitForResources(t, filepath.Join(outDir, "logs.jsonl"), logsResources))
	})
	t.Run("forwarded metrics", func(t *testing.T) {
		assert.Equal(t, want, waitForResources(t, filepath.Join(outDir, "metrics.jsonl"), metricsResources))
	})
	t.Run("RED metrics", func(t *testing.T) {
		assert.Equal(t, want, waitForResources(t, filepath.Join(outDir, "red_metrics.jsonl"), metricsResources))
	})
}

func startCollector(t *testing.T) {
	ctx := logger.WithCtx(context.Background(), logger.GetNop())
	col, err := observecol.GetOtelCollector(ctx)
	require.NoError(t, err)

	runErr := make(chan error, 1)
	go func() { runErr <- col.Run(ctx) }()

	deadline := time.After(30 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	for col.GetState() != otelcol.StateRunning {
		select {
		case err := <-runErr:
			require.FailNow(t, "collector exited during startup", "error: %v", err)
		case <-deadline:
			col.Shutdown()
			require.FailNow(t, "collector did not start within 30s")
		case <-ticker.C:
		}
	}
	t.Cleanup(func() {
		col.Shutdown()
		<-runErr
	})
}

func setResource(res pcommon.Resource, service string, attrs map[string]string) {
	res.Attributes().PutStr("service.name", service)
	for k, v := range attrs {
		res.Attributes().PutStr(k, v)
	}
}

func buildTraces(t *testing.T) []byte {
	td := ptrace.NewTraces()
	now := pcommon.NewTimestampFromTime(time.Now())
	for i, r := range appResources {
		rs := td.ResourceSpans().AppendEmpty()
		setResource(rs.Resource(), r.service, r.attrs)
		span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		span.SetName("GET /")
		span.SetKind(ptrace.SpanKindServer)
		span.SetTraceID(pcommon.TraceID([16]byte{byte(i + 1)}))
		span.SetSpanID(pcommon.SpanID([8]byte{byte(i + 1)}))
		span.SetStartTimestamp(now)
		span.SetEndTimestamp(now)
	}
	body, err := (&ptrace.JSONMarshaler{}).MarshalTraces(td)
	require.NoError(t, err)
	return body
}

func buildLogs(t *testing.T) []byte {
	ld := plog.NewLogs()
	for _, r := range appResources {
		rl := ld.ResourceLogs().AppendEmpty()
		setResource(rl.Resource(), r.service, r.attrs)
		rl.ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("hello")
	}
	body, err := (&plog.JSONMarshaler{}).MarshalLogs(ld)
	require.NoError(t, err)
	return body
}

func buildMetrics(t *testing.T) []byte {
	md := pmetric.NewMetrics()
	for _, r := range appResources {
		rm := md.ResourceMetrics().AppendEmpty()
		setResource(rm.Resource(), r.service, r.attrs)
		m := rm.ScopeMetrics().AppendEmpty().Metrics().AppendEmpty()
		m.SetName("app.requests")
		dp := m.SetEmptyGauge().DataPoints().AppendEmpty()
		dp.SetTimestamp(pcommon.NewTimestampFromTime(time.Now()))
		dp.SetIntValue(1)
	}
	body, err := (&pmetric.JSONMarshaler{}).MarshalMetrics(md)
	require.NoError(t, err)
	return body
}

func postOTLP(t *testing.T, baseURL, path string, body []byte) {
	resp, err := http.Post(baseURL+path, "application/json", bytes.NewReader(body))
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode, "POST %s", path)
}

// resourcesFn decodes one line of file exporter output into the resources it contains.
type resourcesFn func(line []byte) ([]pcommon.Resource, error)

func tracesResources(line []byte) ([]pcommon.Resource, error) {
	td, err := (&ptrace.JSONUnmarshaler{}).UnmarshalTraces(line)
	if err != nil {
		return nil, err
	}
	var out []pcommon.Resource
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		out = append(out, td.ResourceSpans().At(i).Resource())
	}
	return out, nil
}

func logsResources(line []byte) ([]pcommon.Resource, error) {
	ld, err := (&plog.JSONUnmarshaler{}).UnmarshalLogs(line)
	if err != nil {
		return nil, err
	}
	var out []pcommon.Resource
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		out = append(out, ld.ResourceLogs().At(i).Resource())
	}
	return out, nil
}

func metricsResources(line []byte) ([]pcommon.Resource, error) {
	md, err := (&pmetric.JSONUnmarshaler{}).UnmarshalMetrics(line)
	if err != nil {
		return nil, err
	}
	var out []pcommon.Resource
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		out = append(out, md.ResourceMetrics().At(i).Resource())
	}
	return out, nil
}

func attrOrUnset(res pcommon.Resource, key string) string {
	if v, ok := res.Attributes().Get(key); ok {
		return v.AsString()
	}
	return "<unset>"
}

// waitForResources polls a file exporter output file until a resource has been seen for every
// app service, and returns the environment attributes of each service's first resource.
func waitForResources(t *testing.T, path string, decode resourcesFn) map[string]envAttrs {
	got := map[string]envAttrs{}
	require.Eventually(t, func() bool {
		f, err := os.Open(path)
		if err != nil {
			return false
		}
		defer f.Close()
		scanner := bufio.NewScanner(f)
		scanner.Buffer(nil, 10*1024*1024)
		for scanner.Scan() {
			resources, err := decode(scanner.Bytes())
			require.NoError(t, err)
			for _, res := range resources {
				service := attrOrUnset(res, "service.name")
				if _, seen := got[service]; !seen {
					got[service] = envAttrs{Name: attrOrUnset(res, envNameKey), Legacy: attrOrUnset(res, envLegacyKey)}
				}
			}
		}
		for _, r := range appResources {
			if _, seen := got[r.service]; !seen {
				return false
			}
		}
		return true
	}, 15*time.Second, 100*time.Millisecond, "waiting for all services in %s", path)
	return got
}
