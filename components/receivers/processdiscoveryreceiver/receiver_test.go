package processdiscoveryreceiver

import (
	"context"
	"errors"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.uber.org/zap"
)

type staticSource struct{ result ScanResult }

func (s staticSource) Scan(context.Context) (ScanResult, error) { return s.result, nil }
func (s staticSource) Inspect(context.Context, int32) (ProcessSnapshot, error) {
	return s.result.Processes[0], nil
}

type failOnceConsumer struct {
	consumertest.LogsSink
	failed bool
}

func (c *failOnceConsumer) ConsumeLogs(ctx context.Context, logs plog.Logs) error {
	if !c.failed {
		c.failed = true
		return errors.New("temporary downstream failure")
	}
	return c.LogsSink.ConsumeLogs(ctx, logs)
}

func TestBuildLogsSanitizesProcessData(t *testing.T) {
	set := receivertest.NewNopSettings(component.MustNewType("processdiscovery"))
	set.Logger = zap.NewNop()
	event := processEvent{Name: "process.started", Process: snapshot(10, 100, "java")}
	logs := buildLogs(set, []processEvent{event}, nil)
	record := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	serialized := record.Attributes().AsRaw()
	assert.Equal(t, "process.discovery", record.Body().Str())
	assert.Equal(t, "process.started", record.EventName())
	assert.NotContains(t, serialized, "otel.instrumentation")
	assert.False(t, strings.Contains(record.Body().Str(), "/"))
}

func TestBuildLogsFoldsInferredVersionIntoRuntimeVersion(t *testing.T) {
	set := receivertest.NewNopSettings(component.MustNewType("processdiscovery"))
	process := snapshot(10, 100, "go")
	process.InferredVersion = "go1.25.0"
	process.InferredVersionKind = "go_toolchain"
	logs := buildLogs(set, []processEvent{{Name: "process.started", Process: process}}, nil)
	attrs := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	value, ok := attrs.Get("process.runtime.version")
	require.True(t, ok)
	assert.Equal(t, "go1.25.0", value.Str())
}

func TestBuildLogsExplicitRuntimeVersionTakesPrecedence(t *testing.T) {
	set := receivertest.NewNopSettings(component.MustNewType("processdiscovery"))
	process := snapshot(10, 100, "java")
	process.RuntimeVersion = "21.0.1"
	process.InferredVersion = "21.0.0"
	logs := buildLogs(set, []processEvent{{Name: "process.started", Process: process}}, nil)
	attrs := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0).Attributes()
	value, ok := attrs.Get("process.runtime.version")
	require.True(t, ok)
	assert.Equal(t, "21.0.1", value.Str())
}

func TestSnapshotChangeIsNotReportedAsExec(t *testing.T) {
	set := receivertest.NewNopSettings(component.MustNewType("processdiscovery"))
	logs := buildLogs(set, []processEvent{{Name: "process.changed", Process: snapshot(10, 100, "java")}}, nil)
	record := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	assert.Equal(t, "process.changed", record.EventName())
}

func TestBuildLogsAddsHostResourceScope(t *testing.T) {
	set := receivertest.NewNopSettings(component.MustNewType("processdiscovery"))
	logs := buildLogs(set, []processEvent{{Name: "process.started", Process: snapshot(10, 100, "go")}}, map[string]string{
		"host.id": "host-1", "host.arch": "amd64", "os.type": "linux",
	})
	attrs := logs.ResourceLogs().At(0).Resource().Attributes()
	hostID, ok := attrs.Get("host.id")
	require.True(t, ok)
	assert.Equal(t, "host-1", hostID.Str())
}

func TestHostResourceAttributesDoNotUseCollectorIdentityForMountedProcFS(t *testing.T) {
	attrs := hostResourceAttributes(&Config{ProcFSPath: "/host/proc"})
	assert.Equal(t, runtime.GOOS, attrs["os.type"])
	assert.NotContains(t, attrs, "host.id")
	assert.NotContains(t, attrs, "host.arch")
}

func TestHostResourceAttributesUseExplicitObservedHostIdentity(t *testing.T) {
	attrs := hostResourceAttributes(&Config{
		ProcFSPath: "/host/proc", HostID: "observed-host", HostArch: "arm64",
	})
	assert.Equal(t, "observed-host", attrs["host.id"])
	assert.Equal(t, "arm64", attrs["host.arch"])
}

func TestReceiverRetriesStateAfterConsumerFailure(t *testing.T) {
	next := new(failOnceConsumer)
	r := newTestReceiver(t, next)
	r.scanAndEmit(t.Context())
	assert.Empty(t, next.AllLogs())
	r.scanAndEmit(t.Context())
	require.Len(t, next.AllLogs(), 1)
}

func TestLifecycleExitSchedulesOneReconciliationForBurst(t *testing.T) {
	r := newTestReceiver(t, consumertest.NewNop())
	r.applyLifecycleEvent(t.Context(), LifecycleEvent{Type: lifecycleExit, PID: 10})
	first := r.exitReconcile
	require.NotNil(t, first)

	r.applyLifecycleEvent(t.Context(), LifecycleEvent{Type: lifecycleExit, PID: 11})
	assert.Equal(t, first, r.exitReconcile)
}

func newTestReceiver(t *testing.T, next consumer.Logs) *processDiscoveryReceiver {
	t.Helper()
	set := receivertest.NewNopSettings(component.MustNewType("processdiscovery"))
	set.Logger = zap.NewNop()
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{LongLivedCtx: true, ReceiverID: set.ID, ReceiverCreateSettings: set})
	require.NoError(t, err)
	telemetry, err := newReceiverTelemetry(set.TelemetrySettings)
	require.NoError(t, err)
	receiver := &processDiscoveryReceiver{
		cfg:      &Config{CollectionInterval: time.Second, ReportInterval: time.Minute, TerminationGraceScans: 1, MaxTrackedProcesses: 10, Filtering: FilterConfig{MinLifetimeScans: 1}},
		settings: set, nextConsumer: next, obsrecv: obsrecv, telemetry: telemetry,
		source:    staticSource{result: ScanResult{Processes: []ProcessSnapshot{snapshot(10, 100, "java")}, Complete: true}},
		lifecycle: disabledLifecycleSource{}, state: newProcessState(), supported: true, collectionMode: "snapshot_only",
	}
	receiver.references.Store(1)
	receiver.emitter = newLogsEmitter(receiver)
	return receiver
}
