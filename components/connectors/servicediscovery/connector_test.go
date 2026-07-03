package servicediscovery

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap/zaptest"
)

// TestConsumeTraces tests that the connector properly caches service information from traces
func TestConsumeTraces(t *testing.T) {

	// Create a logs sink to capture exported logs
	sink := &consumertest.LogsSink{}

	// Create the connector
	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Create test trace data
	traces := createTestTraces("test-service", "test-namespace", "production", "my-library", "1.0.0")

	// Consume the traces
	ctx := context.Background()
	err = conn.ConsumeTraces(ctx, traces)
	require.NoError(t, err)

	// Verify that the service was cached
	assert.Len(t, conn.opentelemetryServices, 1)
	assert.Equal(t, "test-service", conn.opentelemetryServices[0].ServiceName)
	assert.Equal(t, "test-namespace", conn.opentelemetryServices[0].ServiceNamespace)
	assert.Equal(t, "production", conn.opentelemetryServices[0].Environment)
	assert.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 1)
	assert.Equal(t, "my-library", conn.opentelemetryServices[0].InstrumentationLibraries[0].Name)
	assert.Equal(t, "1.0.0", conn.opentelemetryServices[0].InstrumentationLibraries[0].Version)
	assert.Greater(t, conn.opentelemetryServices[0].InstrumentationLibraries[0].LastSeen, int64(0))
}

// TestConsumeMetrics tests that the connector properly caches service information from metrics
func TestConsumeMetrics(t *testing.T) {
	// Create a logs sink to capture exported logs
	sink := &consumertest.LogsSink{}

	// Create the connector
	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Create test metric data
	metrics := createTestMetrics("metrics-service", "metrics-namespace", "staging", "metrics-lib", "2.0.0")

	// Consume the metrics
	ctx := context.Background()
	err = conn.ConsumeMetrics(ctx, metrics)
	require.NoError(t, err)

	// Verify that the service was cached
	assert.Len(t, conn.opentelemetryServices, 1)
	assert.Equal(t, "metrics-service", conn.opentelemetryServices[0].ServiceName)
	assert.Equal(t, "metrics-namespace", conn.opentelemetryServices[0].ServiceNamespace)
	assert.Equal(t, "staging", conn.opentelemetryServices[0].Environment)
	assert.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 1)
	assert.Equal(t, "metrics-lib", conn.opentelemetryServices[0].InstrumentationLibraries[0].Name)
	assert.Equal(t, "2.0.0", conn.opentelemetryServices[0].InstrumentationLibraries[0].Version)
	assert.Greater(t, conn.opentelemetryServices[0].InstrumentationLibraries[0].LastSeen, int64(0))
}

// TestConsumeLogs tests that the connector properly caches service information from logs
func TestConsumeLogs(t *testing.T) {
	// Create a logs sink to capture exported logs
	sink := &consumertest.LogsSink{}

	// Create the connector
	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Create test log data
	logs := createTestLogs("logs-service", "logs-namespace", "development", "logs-lib", "3.0.0")

	// Consume the logs
	ctx := context.Background()
	err = conn.ConsumeLogs(ctx, logs)
	require.NoError(t, err)

	// Verify that the service was cached
	assert.Len(t, conn.opentelemetryServices, 1)
	assert.Equal(t, "logs-service", conn.opentelemetryServices[0].ServiceName)
	assert.Equal(t, "logs-namespace", conn.opentelemetryServices[0].ServiceNamespace)
	assert.Equal(t, "development", conn.opentelemetryServices[0].Environment)
	assert.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 1)
	assert.Equal(t, "logs-lib", conn.opentelemetryServices[0].InstrumentationLibraries[0].Name)
	assert.Equal(t, "3.0.0", conn.opentelemetryServices[0].InstrumentationLibraries[0].Version)
	assert.Greater(t, conn.opentelemetryServices[0].InstrumentationLibraries[0].LastSeen, int64(0))
}

// TestExportLogs tests that the connector properly exports cached services as logs
func TestExportLogs(t *testing.T) {
	// Create a logs sink to capture exported logs
	sink := &consumertest.LogsSink{}

	// Create the connector
	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)
	require.NotNil(t, conn)

	// Manually populate the cache with test data
	now := time.Now()
	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:      "service-1",
			ServiceNamespace: "namespace-1",
			Environment:      "prod",
			InstrumentationLibraries: []InstrumentationLibrary{
				{
					Name:     "lib-1",
					Version:  "1.0.0",
					LastSeen: now.UnixNano(),
				},
				{
					Name:     "lib-2",
					Version:  "2.0.0",
					LastSeen: now.UnixNano(),
				},
			},
		},
		{
			ServiceName:      "service-2",
			ServiceNamespace: "namespace-2",
			Environment:      "staging",
			InstrumentationLibraries: []InstrumentationLibrary{
				{
					Name:     "lib-3",
					Version:  "3.0.0",
					LastSeen: now.UnixNano(),
				},
			},
		},
	}

	// Set up global export worker state for testing

	// Export logs
	conn.exportLogs()

	// Verify that logs were exported - should be 3 logs (one per library)
	// service-1 has 2 libraries, service-2 has 1 library = 3 total
	allLogs := sink.AllLogs()
	require.Len(t, allLogs, 1)

	logs := allLogs[0]
	assert.Equal(t, 3, logs.ResourceLogs().Len())

	// Verify first library log (service-1, lib-1)
	logRecord1 := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	// Check observe_transform structure
	otVal1, ok := logRecord1.Attributes().Get("observe_transform")
	require.True(t, ok)
	ot1 := otVal1.Map()

	ids1, ok := ot1.Get("identifiers")
	require.True(t, ok)
	sn1, _ := ids1.Map().Get("service.name")
	assert.Equal(t, "service-1", sn1.Str())
	ln1, _ := ids1.Map().Get("instrumentation_library.name")
	assert.Equal(t, "lib-1", ln1.Str())

	kind1, ok := ot1.Get("kind")
	require.True(t, ok)
	assert.Equal(t, "ServiceDiscovery", kind1.Str())

	ctrl1, ok := ot1.Get("control")
	require.True(t, ok)
	et1, _ := ctrl1.Map().Get("eventType")
	assert.Equal(t, "HEARTBEAT", et1.Str())

	// Verify second library log (service-1, lib-2)
	logRecord2 := logs.ResourceLogs().At(1).ScopeLogs().At(0).LogRecords().At(0)
	otVal2, ok := logRecord2.Attributes().Get("observe_transform")
	require.True(t, ok)
	ids2, _ := otVal2.Map().Get("identifiers")
	ln2, _ := ids2.Map().Get("instrumentation_library.name")
	assert.Equal(t, "lib-2", ln2.Str())

	// Verify third library log (service-2, lib-3)
	logRecord3 := logs.ResourceLogs().At(2).ScopeLogs().At(0).LogRecords().At(0)
	otVal3, ok := logRecord3.Attributes().Get("observe_transform")
	require.True(t, ok)
	ids3, _ := otVal3.Map().Get("identifiers")
	sn3, _ := ids3.Map().Get("service.name")
	assert.Equal(t, "service-2", sn3.Str())
	ln3, _ := ids3.Map().Get("instrumentation_library.name")
	assert.Equal(t, "lib-3", ln3.Str())
}

// TestStartAndShutdown tests the lifecycle of the connector
func TestStartAndShutdown(t *testing.T) {
	sink := &consumertest.LogsSink{}

	cfg := &Config{
		LogExportInterval: 100 * time.Millisecond,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Start the connector
	ctx := context.Background()
	err = conn.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Verify global ticker is running (only if this is the first connector instance)
	// Note: We can't reliably test this since sync.Once may have already been called

	// Shutdown the connector
	err = conn.Shutdown(ctx)
	require.NoError(t, err)
}

// TestPeriodicExport tests that logs are exported periodically
func TestPeriodicExport(t *testing.T) {
	sink := &consumertest.LogsSink{}

	// Use a short interval for testing
	cfg := &Config{
		LogExportInterval: 50 * time.Millisecond,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Populate cache with test data
	now := time.Now()
	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:      "test-service",
			ServiceNamespace: "test-namespace",
			Environment:      "test",
			InstrumentationLibraries: []InstrumentationLibrary{
				{
					Name:     "test-lib",
					Version:  "1.0.0",
					LastSeen: now.UnixNano(),
				},
			},
		},
	}

	// Start the connector
	ctx := context.Background()
	err = conn.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Wait for at least one export cycle
	time.Sleep(150 * time.Millisecond)

	// Shutdown
	err = conn.Shutdown(ctx)
	require.NoError(t, err)

	// Verify that logs were exported at least once
	allLogs := sink.AllLogs()
	assert.GreaterOrEqual(t, len(allLogs), 1)

	if len(allLogs) > 0 {
		logs := allLogs[0]
		assert.Equal(t, 1, logs.ResourceLogs().Len())
		logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
		otVal, ok := logRecord.Attributes().Get("observe_transform")
		require.True(t, ok)
		kind, _ := otVal.Map().Get("kind")
		assert.Equal(t, "ServiceDiscovery", kind.Str())
	}
}

// TestExportLogsWithNilConsumer tests that exportLogs handles nil consumer gracefully
func TestExportLogsWithNilConsumer(t *testing.T) {
	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, nil)
	require.NoError(t, err)

	// Populate cache
	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:      "test-service",
			ServiceNamespace: "test-namespace",
			Environment:      "test",
			InstrumentationLibraries: []InstrumentationLibrary{
				{
					Name:     "test-lib",
					Version:  "1.0.0",
					LastSeen: time.Now().UnixNano(),
				},
			},
		},
	}

	// Set up global export worker state for testing

	// Should not panic
	conn.exportLogs()
}

// TestCapabilities tests the Capabilities method
func TestCapabilities(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	caps := conn.Capabilities()
	assert.False(t, caps.MutatesData)
}

// TestExportLogsWithEmptyCache tests that exportLogs handles empty cache gracefully
func TestExportLogsWithEmptyCache(t *testing.T) {
	sink := &consumertest.LogsSink{}

	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Export logs with empty cache
	conn.exportLogs()

	// Verify no logs were exported
	allLogs := sink.AllLogs()
	assert.Len(t, allLogs, 0)
}

// TestConsumeTracesMultipleServices tests consuming traces from multiple services
func TestConsumeTracesMultipleServices(t *testing.T) {
	sink := &consumertest.LogsSink{}

	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Consume traces from multiple services
	ctx := context.Background()
	traces1 := createTestTraces("service-1", "ns-1", "prod", "lib-1", "1.0.0")
	err = conn.ConsumeTraces(ctx, traces1)
	require.NoError(t, err)

	traces2 := createTestTraces("service-2", "ns-2", "staging", "lib-2", "2.0.0")
	err = conn.ConsumeTraces(ctx, traces2)
	require.NoError(t, err)

	// Verify both services were cached
	assert.Len(t, conn.opentelemetryServices, 2)
	assert.Equal(t, "service-1", conn.opentelemetryServices[0].ServiceName)
	assert.Equal(t, "service-2", conn.opentelemetryServices[1].ServiceName)
}

// TestConsumeTracesWithMissingAttributes tests handling of traces with missing attributes
func TestConsumeTracesWithMissingAttributes(t *testing.T) {
	sink := &consumertest.LogsSink{}

	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Create traces with minimal attributes
	traces := ptrace.NewTraces()
	resourceSpan := traces.ResourceSpans().AppendEmpty()
	attrs := resourceSpan.Resource().Attributes()
	attrs.PutStr("service.name", "minimal-service")
	// Missing other attributes

	span := resourceSpan.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("test-span")

	// Should not panic
	ctx := context.Background()
	err = conn.ConsumeTraces(ctx, traces)
	require.NoError(t, err)

	// Verify service was cached (with empty values for missing attributes)
	assert.Len(t, conn.opentelemetryServices, 1)
	assert.Equal(t, "minimal-service", conn.opentelemetryServices[0].ServiceName)
}

// TestExportLogsWithEmptyInstrumentationLibraries tests exporting services with no libraries
func TestExportLogsWithEmptyInstrumentationLibraries(t *testing.T) {
	sink := &consumertest.LogsSink{}

	cfg := &Config{
		LogExportInterval: 1 * time.Hour,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Populate cache with service that has no instrumentation libraries
	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:              "service-no-libs",
			ServiceNamespace:         "namespace",
			Environment:              "prod",
			InstrumentationLibraries: []InstrumentationLibrary{},
		},
	}

	// Export logs
	conn.exportLogs()

	// Verify NO logs were exported (since there are no libraries)
	// With the new format, we export one log per library, so 0 libraries = 0 logs
	allLogs := sink.AllLogs()
	assert.Len(t, allLogs, 0)
}

// TestIntegrationEndToEnd tests the full flow from consuming traces to exporting logs
func TestIntegrationEndToEnd(t *testing.T) {
	sink := &consumertest.LogsSink{}

	cfg := &Config{
		LogExportInterval: 100 * time.Millisecond,
	}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Start the connector
	ctx := context.Background()
	err = conn.Start(ctx, componenttest.NewNopHost())
	require.NoError(t, err)

	// Consume some traces
	traces := createTestTraces("integration-service", "integration-ns", "test", "integration-lib", "1.0.0")
	err = conn.ConsumeTraces(ctx, traces)
	require.NoError(t, err)

	// Wait for export cycle
	time.Sleep(200 * time.Millisecond)

	// Shutdown
	err = conn.Shutdown(ctx)
	require.NoError(t, err)

	// Verify logs were exported
	allLogs := sink.AllLogs()
	assert.GreaterOrEqual(t, len(allLogs), 1)

	if len(allLogs) > 0 {
		logs := allLogs[0]
		assert.GreaterOrEqual(t, logs.ResourceLogs().Len(), 1)

		logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

		// Verify observe_transform identifiers
		otVal, ok := logRecord.Attributes().Get("observe_transform")
		require.True(t, ok)
		ids, _ := otVal.Map().Get("identifiers")
		sn, _ := ids.Map().Get("service.name")
		assert.Equal(t, "integration-service", sn.Str())
		ln, _ := ids.Map().Get("instrumentation_library.name")
		assert.Equal(t, "integration-lib", ln.Str())
		lv, _ := ids.Map().Get("instrumentation_library.version")
		assert.Equal(t, "1.0.0", lv.Str())

	}
}

// createTestTraces creates test trace data with the specified attributes
func createTestTraces(serviceName, serviceNamespace, environment, libName, libVersion string) ptrace.Traces {
	traces := ptrace.NewTraces()
	resourceSpan := traces.ResourceSpans().AppendEmpty()

	// Set resource attributes
	attrs := resourceSpan.Resource().Attributes()
	attrs.PutStr("service.name", serviceName)
	attrs.PutStr("service.namespace", serviceNamespace)
	attrs.PutStr("deployment.environment", environment)

	// Add a scope span with instrumentation library info
	scopeSpan := resourceSpan.ScopeSpans().AppendEmpty()
	scope := scopeSpan.Scope()
	scope.SetName(libName)
	scope.SetVersion(libVersion)

	// Add a span
	span := scopeSpan.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}))
	span.SetSpanID(pcommon.SpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8}))

	return traces
}

// createTestTracesWithSpanKind creates trace data with a specific service version and span kind
func createTestTracesWithSpanKind(serviceName, serviceNamespace, serviceVersion, environment, libName, libVersion string, kind ptrace.SpanKind) ptrace.Traces {
	traces := ptrace.NewTraces()
	resourceSpan := traces.ResourceSpans().AppendEmpty()

	attrs := resourceSpan.Resource().Attributes()
	attrs.PutStr("service.name", serviceName)
	attrs.PutStr("service.namespace", serviceNamespace)
	if serviceVersion != "" {
		attrs.PutStr("service.version", serviceVersion)
	}
	attrs.PutStr("deployment.environment", environment)

	scopeSpan := resourceSpan.ScopeSpans().AppendEmpty()
	scope := scopeSpan.Scope()
	scope.SetName(libName)
	scope.SetVersion(libVersion)

	span := scopeSpan.Spans().AppendEmpty()
	span.SetName("test-span")
	span.SetKind(kind)

	return traces
}

// createTestMetrics creates test metric data with the specified attributes
func createTestMetrics(serviceName, serviceNamespace, environment, libName, libVersion string) pmetric.Metrics {
	metrics := pmetric.NewMetrics()
	resourceMetric := metrics.ResourceMetrics().AppendEmpty()

	// Set resource attributes
	attrs := resourceMetric.Resource().Attributes()
	attrs.PutStr("service.name", serviceName)
	attrs.PutStr("service.namespace", serviceNamespace)
	attrs.PutStr("deployment.environment", environment)

	// Add a scope metric with instrumentation library info
	scopeMetric := resourceMetric.ScopeMetrics().AppendEmpty()
	scope := scopeMetric.Scope()
	scope.SetName(libName)
	scope.SetVersion(libVersion)

	// Add a metric
	metric := scopeMetric.Metrics().AppendEmpty()
	metric.SetName("test-metric")
	metric.SetEmptyGauge()

	return metrics
}

// createTestLogs creates test log data with the specified attributes
func createTestLogs(serviceName, serviceNamespace, environment, libName, libVersion string) plog.Logs {
	logs := plog.NewLogs()
	resourceLog := logs.ResourceLogs().AppendEmpty()

	// Set resource attributes
	attrs := resourceLog.Resource().Attributes()
	attrs.PutStr("service.name", serviceName)
	attrs.PutStr("service.namespace", serviceNamespace)
	attrs.PutStr("deployment.environment", environment)

	// Add a scope log with instrumentation library info
	scopeLog := resourceLog.ScopeLogs().AppendEmpty()
	scope := scopeLog.Scope()
	scope.SetName(libName)
	scope.SetVersion(libVersion)

	// Add a log record
	logRecord := scopeLog.LogRecords().AppendEmpty()
	logRecord.Body().SetStr("test log message")

	return logs
}

// TestServiceDeduplication tests that duplicate services are not created
func TestServiceDeduplication(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Consume the same service twice
	traces1 := createTestTraces("test-service", "test-namespace", "production", "lib1", "1.0.0")
	traces2 := createTestTraces("test-service", "test-namespace", "production", "lib2", "2.0.0")

	ctx := context.Background()
	err = conn.ConsumeTraces(ctx, traces1)
	require.NoError(t, err)
	err = conn.ConsumeTraces(ctx, traces2)
	require.NoError(t, err)

	// Should have only one service with two libraries
	assert.Len(t, conn.opentelemetryServices, 1)
	assert.Equal(t, "test-service", conn.opentelemetryServices[0].ServiceName)
	assert.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 2)
}

// TestLibraryVersionUpdate tests that different versions of the same library are tracked separately
func TestLibraryVersionUpdate(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Consume the same library with different versions
	traces1 := createTestTraces("test-service", "test-namespace", "production", "my-lib", "1.0.0")
	traces2 := createTestTraces("test-service", "test-namespace", "production", "my-lib", "2.0.0")

	ctx := context.Background()
	err = conn.ConsumeTraces(ctx, traces1)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure different timestamps

	err = conn.ConsumeTraces(ctx, traces2)
	require.NoError(t, err)

	// Should have one service with two library versions
	assert.Len(t, conn.opentelemetryServices, 1)
	assert.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 2)

	// Verify both versions are present
	libs := conn.opentelemetryServices[0].InstrumentationLibraries
	versions := make(map[string]bool)
	for _, lib := range libs {
		assert.Equal(t, "my-lib", lib.Name)
		versions[lib.Version] = true
	}
	assert.True(t, versions["1.0.0"])
	assert.True(t, versions["2.0.0"])
}

// TestLibraryLastSeenUpdate tests that lastSeen is updated for existing library versions
func TestLibraryLastSeenUpdate(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Consume the same library twice
	traces1 := createTestTraces("test-service", "test-namespace", "production", "my-lib", "1.0.0")

	ctx := context.Background()
	err = conn.ConsumeTraces(ctx, traces1)
	require.NoError(t, err)

	firstLastSeen := conn.opentelemetryServices[0].InstrumentationLibraries[0].LastSeen

	time.Sleep(10 * time.Millisecond) // Ensure different timestamp

	traces2 := createTestTraces("test-service", "test-namespace", "production", "my-lib", "1.0.0")
	err = conn.ConsumeTraces(ctx, traces2)
	require.NoError(t, err)

	// Should still have one service with one library
	assert.Len(t, conn.opentelemetryServices, 1)
	assert.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 1)

	// LastSeen should be updated
	secondLastSeen := conn.opentelemetryServices[0].InstrumentationLibraries[0].LastSeen
	assert.Greater(t, secondLastSeen, firstLastSeen)
}

// TestMultipleServicesWithSameLibrary tests that different services can have the same library
func TestMultipleServicesWithSameLibrary(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// Consume traces from different services with the same library
	traces1 := createTestTraces("service-1", "namespace-1", "production", "common-lib", "1.0.0")
	traces2 := createTestTraces("service-2", "namespace-2", "production", "common-lib", "1.0.0")

	ctx := context.Background()
	err = conn.ConsumeTraces(ctx, traces1)
	require.NoError(t, err)
	err = conn.ConsumeTraces(ctx, traces2)
	require.NoError(t, err)

	// Should have two services, each with the same library
	assert.Len(t, conn.opentelemetryServices, 2)
	assert.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 1)
	assert.Len(t, conn.opentelemetryServices[1].InstrumentationLibraries, 1)
	assert.Equal(t, "common-lib", conn.opentelemetryServices[0].InstrumentationLibraries[0].Name)
	assert.Equal(t, "common-lib", conn.opentelemetryServices[1].InstrumentationLibraries[0].Name)
}

// TestSpanKindDifferentiation tests that the same library seen with different span kinds
// is tracked as separate entries
func TestSpanKindDifferentiation(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	ctx := context.Background()
	serverTraces := createTestTracesWithSpanKind("svc", "ns", "1.0.0", "prod", "lib", "1.0.0", ptrace.SpanKindServer)
	clientTraces := createTestTracesWithSpanKind("svc", "ns", "1.0.0", "prod", "lib", "1.0.0", ptrace.SpanKindClient)

	require.NoError(t, conn.ConsumeTraces(ctx, serverTraces))
	require.NoError(t, conn.ConsumeTraces(ctx, clientTraces))

	// Same service, same lib/version, but two distinct span kinds => two library entries
	require.Len(t, conn.opentelemetryServices, 1)
	require.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 2)

	kinds := make(map[string]bool)
	for _, lib := range conn.opentelemetryServices[0].InstrumentationLibraries {
		kinds[lib.SpanKind] = true
	}
	assert.True(t, kinds["Server"])
	assert.True(t, kinds["Client"])
}

// TestSpanKindDeduplication tests that repeated spans with the same kind collapse into one entry
func TestSpanKindDeduplication(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	ctx := context.Background()
	traces1 := createTestTracesWithSpanKind("svc", "ns", "1.0.0", "prod", "lib", "1.0.0", ptrace.SpanKindServer)
	traces2 := createTestTracesWithSpanKind("svc", "ns", "1.0.0", "prod", "lib", "1.0.0", ptrace.SpanKindServer)

	require.NoError(t, conn.ConsumeTraces(ctx, traces1))
	require.NoError(t, conn.ConsumeTraces(ctx, traces2))

	require.Len(t, conn.opentelemetryServices, 1)
	require.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 1)
	assert.Equal(t, "Server", conn.opentelemetryServices[0].InstrumentationLibraries[0].SpanKind)
}

// TestServiceVersionDifferentiation tests that different service versions produce distinct services
func TestServiceVersionDifferentiation(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	ctx := context.Background()
	v1 := createTestTracesWithSpanKind("svc", "ns", "1.0.0", "prod", "lib", "1.0.0", ptrace.SpanKindServer)
	v2 := createTestTracesWithSpanKind("svc", "ns", "2.0.0", "prod", "lib", "1.0.0", ptrace.SpanKindServer)

	require.NoError(t, conn.ConsumeTraces(ctx, v1))
	require.NoError(t, conn.ConsumeTraces(ctx, v2))

	require.Len(t, conn.opentelemetryServices, 2)
	versions := make(map[string]bool)
	for _, svc := range conn.opentelemetryServices {
		versions[svc.ServiceVersion] = true
	}
	assert.True(t, versions["1.0.0"])
	assert.True(t, versions["2.0.0"])
}

// TestExportLogsIncludesVersionAndSpanKind tests that the new key fields appear in identifiers
func TestExportLogsIncludesVersionAndSpanKind(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	now := time.Now()
	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:      "svc",
			ServiceNamespace: "ns",
			ServiceVersion:   "1.2.3",
			Environment:      "prod",
			InstrumentationLibraries: []InstrumentationLibrary{
				{Name: "lib", Version: "1.0.0", SpanKind: "Server", LastSeen: now.UnixNano()},
			},
		},
	}

	conn.exportLogs()

	allLogs := sink.AllLogs()
	require.Len(t, allLogs, 1)
	logRecord := allLogs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	otVal, ok := logRecord.Attributes().Get("observe_transform")
	require.True(t, ok)
	ids, _ := otVal.Map().Get("identifiers")

	version, ok := ids.Map().Get("service.version")
	require.True(t, ok)
	assert.Equal(t, "1.2.3", version.Str())

	spanKind, ok := ids.Map().Get("span.kind")
	require.True(t, ok)
	assert.Equal(t, "Server", spanKind.Str())
}

// TestExportLogsOmitsEmptyVersionAndSpanKind tests that empty service version / span kind
// (e.g. from metrics or logs) are omitted from identifiers and body
func TestExportLogsOmitsEmptyVersionAndSpanKind(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:      "svc",
			ServiceNamespace: "ns",
			Environment:      "prod",
			InstrumentationLibraries: []InstrumentationLibrary{
				{Name: "lib", Version: "1.0.0", LastSeen: time.Now().UnixNano()},
			},
		},
	}

	conn.exportLogs()

	logRecord := sink.AllLogs()[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	otVal, ok := logRecord.Attributes().Get("observe_transform")
	require.True(t, ok)
	ids, _ := otVal.Map().Get("identifiers")

	_, ok = ids.Map().Get("service.version")
	assert.False(t, ok)
	_, ok = ids.Map().Get("span.kind")
	assert.False(t, ok)
}

// TestExportLogsIncludesSampleSpan tests that the entire sample span is embedded in facets
func TestExportLogsIncludesSampleSpan(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	sampleSpan := ptrace.NewSpan()
	sampleSpan.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	sampleSpan.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
	sampleSpan.SetParentSpanID([8]byte{0, 0, 0, 0, 0, 0, 0, 1})
	sampleSpan.SetName("GET /api/users")
	sampleSpan.SetKind(ptrace.SpanKindServer)
	sampleSpan.SetStartTimestamp(1000000000)
	sampleSpan.SetEndTimestamp(2000000000)
	sampleSpan.Status().SetCode(ptrace.StatusCodeOk)
	sampleSpan.Attributes().PutStr("http.method", "GET")
	sampleSpan.Attributes().PutStr("http.target", "/api/users")
	sampleSpan.Attributes().PutInt("http.status_code", 200)

	sampleResourceAttrs := pcommon.NewMap()
	sampleResourceAttrs.PutStr("service.name", "user-service")
	sampleResourceAttrs.PutStr("service.namespace", "backend")
	sampleResourceAttrs.PutStr("service.version", "2.0.0")
	sampleResourceAttrs.PutStr("deployment.environment.name", "prod")
	sampleResourceAttrs.PutStr("host.name", "worker-01")

	now := time.Now()
	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:      "user-service",
			ServiceNamespace: "backend",
			ServiceVersion:   "2.0.0",
			Environment:      "prod",
			InstrumentationLibraries: []InstrumentationLibrary{
				{
					Name:                "go.opentelemetry.io/contrib/instrumentation/net/http",
					Version:             "0.44.0",
					SpanKind:            "Server",
					LastSeen:            now.UnixNano(),
					HasSampleSpan:       true,
					SampleSpan:          sampleSpan,
					SampleResourceAttrs: sampleResourceAttrs,
				},
			},
		},
	}

	conn.exportLogs()

	allLogs := sink.AllLogs()
	require.Len(t, allLogs, 1)
	logRecord := allLogs[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	otVal, ok := logRecord.Attributes().Get("observe_transform")
	require.True(t, ok)
	facets, ok := otVal.Map().Get("facets")
	require.True(t, ok)

	sampleMapVal, ok := facets.Map().Get("sample_span")
	require.True(t, ok, "facets should contain sample_span")
	sm := sampleMapVal.Map()

	traceID, _ := sm.Get("trace_id")
	assert.Equal(t, "0102030405060708090a0b0c0d0e0f10", traceID.Str())

	spanID, _ := sm.Get("span_id")
	assert.Equal(t, "0102030405060708", spanID.Str())

	parentSpanID, _ := sm.Get("parent_span_id")
	assert.Equal(t, "0000000000000001", parentSpanID.Str())

	name, _ := sm.Get("name")
	assert.Equal(t, "GET /api/users", name.Str())

	kind, _ := sm.Get("kind")
	assert.Equal(t, "Server", kind.Str())

	startTime, _ := sm.Get("start_time_unix_nano")
	assert.Equal(t, int64(1000000000), startTime.Int())

	endTime, _ := sm.Get("end_time_unix_nano")
	assert.Equal(t, int64(2000000000), endTime.Int())

	statusCode, _ := sm.Get("status.code")
	assert.Equal(t, "Ok", statusCode.Str())

	attrVal, ok := sm.Get("attributes")
	require.True(t, ok)
	attrs := attrVal.Map()
	httpMethod, _ := attrs.Get("http.method")
	assert.Equal(t, "GET", httpMethod.Str())
	httpTarget, _ := attrs.Get("http.target")
	assert.Equal(t, "/api/users", httpTarget.Str())
	httpStatus, _ := attrs.Get("http.status_code")
	assert.Equal(t, int64(200), httpStatus.Int())

	resAttrVal, ok := sm.Get("resource_attributes")
	require.True(t, ok, "sample_span should contain resource_attributes")
	resAttrs := resAttrVal.Map()
	hostName, _ := resAttrs.Get("host.name")
	assert.Equal(t, "worker-01", hostName.Str())
	resSvcName, _ := resAttrs.Get("service.name")
	assert.Equal(t, "user-service", resSvcName.Str())
}

// TestExportLogsNoSampleSpanWhenAbsent verifies no sample_span when HasSampleSpan is false
func TestExportLogsNoSampleSpanWhenAbsent(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:      "svc",
			ServiceNamespace: "ns",
			Environment:      "prod",
			InstrumentationLibraries: []InstrumentationLibrary{
				{Name: "lib", Version: "1.0.0", LastSeen: time.Now().UnixNano()},
			},
		},
	}

	conn.exportLogs()

	logRecord := sink.AllLogs()[0].ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)
	otVal, ok := logRecord.Attributes().Get("observe_transform")
	require.True(t, ok)
	facets, ok := otVal.Map().Get("facets")
	require.True(t, ok)

	_, ok = facets.Map().Get("sample_span")
	assert.False(t, ok, "sample_span should not be present when HasSampleSpan is false")
}

// TestSampleSpanWriteOnce verifies that once a sample span is captured, subsequent spans don't overwrite it
func TestSampleSpanWriteOnce(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	// First trace with span "GET /users"
	traces1 := ptrace.NewTraces()
	rs1 := traces1.ResourceSpans().AppendEmpty()
	rs1.Resource().Attributes().PutStr("service.name", "svc")
	rs1.Resource().Attributes().PutStr("service.namespace", "ns")
	rs1.Resource().Attributes().PutStr("deployment.environment.name", "prod")
	ss1 := rs1.ScopeSpans().AppendEmpty()
	ss1.Scope().SetName("lib")
	ss1.Scope().SetVersion("1.0.0")
	span1 := ss1.Spans().AppendEmpty()
	span1.SetName("GET /users")
	span1.SetTraceID([16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16})
	span1.SetSpanID([8]byte{1, 2, 3, 4, 5, 6, 7, 8})
	span1.SetKind(ptrace.SpanKindServer)

	err = conn.ConsumeTraces(context.Background(), traces1)
	require.NoError(t, err)

	// Second trace with span "POST /orders" — same key
	traces2 := ptrace.NewTraces()
	rs2 := traces2.ResourceSpans().AppendEmpty()
	rs2.Resource().Attributes().PutStr("service.name", "svc")
	rs2.Resource().Attributes().PutStr("service.namespace", "ns")
	rs2.Resource().Attributes().PutStr("deployment.environment.name", "prod")
	ss2 := rs2.ScopeSpans().AppendEmpty()
	ss2.Scope().SetName("lib")
	ss2.Scope().SetVersion("1.0.0")
	span2 := ss2.Spans().AppendEmpty()
	span2.SetName("POST /orders")
	span2.SetTraceID([16]byte{16, 15, 14, 13, 12, 11, 10, 9, 8, 7, 6, 5, 4, 3, 2, 1})
	span2.SetSpanID([8]byte{8, 7, 6, 5, 4, 3, 2, 1})
	span2.SetKind(ptrace.SpanKindServer)

	err = conn.ConsumeTraces(context.Background(), traces2)
	require.NoError(t, err)

	// The sample should still be the first span
	require.Len(t, conn.opentelemetryServices, 1)
	lib := conn.opentelemetryServices[0].InstrumentationLibraries[0]
	assert.True(t, lib.HasSampleSpan)
	assert.Equal(t, "GET /users", lib.SampleSpan.Name())
	assert.Equal(t, "0102030405060708090a0b0c0d0e0f10", lib.SampleSpan.TraceID().String())
}

// TestEvictExpiredEntries verifies that entries with LastSeen older than 7 days are removed
func TestEvictExpiredEntries(t *testing.T) {
	sink := &consumertest.LogsSink{}
	cfg := &Config{LogExportInterval: 1 * time.Hour}
	logger := zaptest.NewLogger(t)
	conn, err := newConnector(logger, cfg, sink)
	require.NoError(t, err)

	now := time.Now()
	eightDaysAgo := now.Add(-8 * 24 * time.Hour).UnixNano()
	oneDayAgo := now.Add(-1 * 24 * time.Hour).UnixNano()

	conn.opentelemetryServices = []OpentelemetryService{
		{
			ServiceName:      "expired-svc",
			ServiceNamespace: "ns",
			Environment:      "prod",
			InstrumentationLibraries: []InstrumentationLibrary{
				{Name: "lib-old", Version: "1.0.0", LastSeen: eightDaysAgo},
			},
		},
		{
			ServiceName:      "active-svc",
			ServiceNamespace: "ns",
			Environment:      "prod",
			InstrumentationLibraries: []InstrumentationLibrary{
				{Name: "lib-new", Version: "1.0.0", LastSeen: oneDayAgo},
				{Name: "lib-expired", Version: "2.0.0", LastSeen: eightDaysAgo},
			},
		},
	}

	conn.evictExpiredEntries()

	// expired-svc should be gone entirely (all its libs expired)
	// active-svc should remain with only lib-new
	require.Len(t, conn.opentelemetryServices, 1)
	assert.Equal(t, "active-svc", conn.opentelemetryServices[0].ServiceName)
	require.Len(t, conn.opentelemetryServices[0].InstrumentationLibraries, 1)
	assert.Equal(t, "lib-new", conn.opentelemetryServices[0].InstrumentationLibraries[0].Name)
}
