package servicediscovery

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

const entryTTL = 7 * 24 * time.Hour

type InstrumentationLibrary struct {
	Name                string
	Version             string
	SpanKind            string
	LastSeen            int64
	HasSampleSpan       bool
	SampleSpan          ptrace.Span
	SampleResourceAttrs pcommon.Map
}

type OpentelemetryService struct {
	Environment              string
	ServiceName              string
	ServiceNamespace         string
	ServiceVersion           string
	TelemetrySdkLanguage     string
	TelemetrySdkVersion      string
	ProcessRuntimeName       string
	ProcessRuntimeVersion    string
	InstrumentationLibraries []InstrumentationLibrary
}

// schema for connector
type serviceDiscoveryConnector struct {
	config                *Config
	logsConsumer          consumer.Logs
	logger                *zap.Logger
	opentelemetryServices []OpentelemetryService
	servicesMu            sync.RWMutex
	logExportTicker       *time.Ticker

	// Include these parameters if a specific implementation for the Start and Shutdown function are not needed
	component.StartFunc
	component.ShutdownFunc

	shutdownCh chan struct{}
	wg         sync.WaitGroup
}

// newConnector is a function to create a new connector
func newConnector(logger *zap.Logger, config component.Config, nextConsumer consumer.Logs) (*serviceDiscoveryConnector, error) {
	logger.Info("Building service discovery connector")
	cfg := config.(*Config)

	return &serviceDiscoveryConnector{
		config:                cfg,
		logger:                logger,
		logsConsumer:          nextConsumer,
		opentelemetryServices: []OpentelemetryService{},
		shutdownCh:            make(chan struct{}),
	}, nil
}

// Capabilities implements the consumer interface.
func (c *serviceDiscoveryConnector) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: false}
}

// findOrCreateService finds an existing service or creates a new one
// Must be called with lock held
func (c *serviceDiscoveryConnector) findOrCreateService(serviceName, serviceNamespace, serviceVersion, environment, sdkLanguage, sdkVersion, runtimeName, runtimeVersion string) *OpentelemetryService {
	// Look for existing service
	for i := range c.opentelemetryServices {
		svc := &c.opentelemetryServices[i]
		if svc.ServiceName == serviceName &&
			svc.ServiceNamespace == serviceNamespace &&
			svc.ServiceVersion == serviceVersion &&
			svc.Environment == environment {
			c.logger.Debug("Found existing service",
				zap.String("service.name", serviceName),
				zap.Int("current_libraries", len(svc.InstrumentationLibraries)))

			// Update SDK and runtime info if they've changed (e.g., service restarted with new version)
			if sdkLanguage != "" && svc.TelemetrySdkLanguage != sdkLanguage {
				svc.TelemetrySdkLanguage = sdkLanguage
			}
			if sdkVersion != "" && svc.TelemetrySdkVersion != sdkVersion {
				svc.TelemetrySdkVersion = sdkVersion
			}
			if runtimeName != "" && svc.ProcessRuntimeName != runtimeName {
				svc.ProcessRuntimeName = runtimeName
			}
			if runtimeVersion != "" && svc.ProcessRuntimeVersion != runtimeVersion {
				svc.ProcessRuntimeVersion = runtimeVersion
			}

			return svc
		}
	}

	// Service not found, create new one
	c.logger.Debug("Creating new service entry",
		zap.String("service.name", serviceName),
		zap.String("service.namespace", serviceNamespace),
		zap.String("service.version", serviceVersion),
		zap.String("deployment.environment", environment),
		zap.String("telemetry.sdk.language", sdkLanguage),
		zap.String("telemetry.sdk.version", sdkVersion),
		zap.String("process.runtime.name", runtimeName),
		zap.String("process.runtime.version", runtimeVersion))

	newService := OpentelemetryService{
		Environment:              environment,
		ServiceName:              serviceName,
		ServiceNamespace:         serviceNamespace,
		ServiceVersion:           serviceVersion,
		TelemetrySdkLanguage:     sdkLanguage,
		TelemetrySdkVersion:      sdkVersion,
		ProcessRuntimeName:       runtimeName,
		ProcessRuntimeVersion:    runtimeVersion,
		InstrumentationLibraries: []InstrumentationLibrary{},
	}
	c.opentelemetryServices = append(c.opentelemetryServices, newService)
	return &c.opentelemetryServices[len(c.opentelemetryServices)-1]
}

// updateInstrumentationLibrary updates or adds an instrumentation library to a service
// Must be called with servicesMu lock held
func (c *serviceDiscoveryConnector) updateInstrumentationLibrary(svc *OpentelemetryService, libName, libVersion, spanKind string, sampleSpan ptrace.Span, resourceAttrs pcommon.Map, lastSeen int64) {
	// Look for existing library with same name, version, and span kind
	for i := range svc.InstrumentationLibraries {
		lib := &svc.InstrumentationLibraries[i]
		if lib.Name == libName && lib.Version == libVersion && lib.SpanKind == spanKind {
			c.logger.Debug("Updating existing library lastSeen",
				zap.String("service.name", svc.ServiceName),
				zap.String("library.name", libName),
				zap.String("library.version", libVersion),
				zap.String("span.kind", spanKind))
			lib.LastSeen = lastSeen
			if !lib.HasSampleSpan && !sampleSpan.TraceID().IsEmpty() {
				lib.SampleSpan = ptrace.NewSpan()
				sampleSpan.CopyTo(lib.SampleSpan)
				lib.SampleResourceAttrs = pcommon.NewMap()
				resourceAttrs.CopyTo(lib.SampleResourceAttrs)
				lib.HasSampleSpan = true
			}
			return
		}
	}

	c.logger.Debug("Adding new library to service",
		zap.String("service.name", svc.ServiceName),
		zap.String("library.name", libName),
		zap.String("library.version", libVersion),
		zap.String("span.kind", spanKind))

	newLib := InstrumentationLibrary{
		Name:     libName,
		Version:  libVersion,
		SpanKind: spanKind,
		LastSeen: lastSeen,
	}
	if !sampleSpan.TraceID().IsEmpty() {
		newLib.SampleSpan = ptrace.NewSpan()
		sampleSpan.CopyTo(newLib.SampleSpan)
		newLib.SampleResourceAttrs = pcommon.NewMap()
		resourceAttrs.CopyTo(newLib.SampleResourceAttrs)
		newLib.HasSampleSpan = true
	}
	svc.InstrumentationLibraries = append(svc.InstrumentationLibraries, newLib)
}

func (c *serviceDiscoveryConnector) Start(ctx context.Context, host component.Host) error {
	c.logger.Info("Starting service discovery connector")

	// Start log export worker
	c.logExportTicker = time.NewTicker(c.config.LogExportInterval)
	c.wg.Add(1)
	go c.logExportWorker()

	return nil
}

func (c *serviceDiscoveryConnector) Shutdown(ctx context.Context) error {
	c.logger.Info("Shutting down service discovery connector")

	// Stop the ticker
	if c.logExportTicker != nil {
		c.logExportTicker.Stop()
	}

	// Signal shutdown to worker
	close(c.shutdownCh)

	// Wait for worker to finish
	c.wg.Wait()

	return nil
}

// ConsumeTraces method is called for each instance of a trace sent to the connector
func (c *serviceDiscoveryConnector) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	// loop through the levels of spans of the one trace consumed
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		resourceSpan := td.ResourceSpans().At(i)

		// Extract resource attributes
		attrs := resourceSpan.Resource().Attributes()
		serviceName, _ := attrs.Get("service.name")
		serviceNamespace, _ := attrs.Get("service.namespace")
		serviceVersion, _ := attrs.Get("service.version")
		deploymentEnvironment, _ := attrs.Get("deployment.environment")
		sdkLanguage, _ := attrs.Get("telemetry.sdk.language")
		sdkVersion, _ := attrs.Get("telemetry.sdk.version")
		runtimeName, _ := attrs.Get("process.runtime.name")
		runtimeVersion, _ := attrs.Get("process.runtime.version")

		now := time.Now()

		c.servicesMu.Lock()
		// Find or create the service
		svc := c.findOrCreateService(
			serviceName.Str(),
			serviceNamespace.Str(),
			serviceVersion.Str(),
			deploymentEnvironment.Str(),
			sdkLanguage.Str(),
			sdkVersion.Str(),
			runtimeName.Str(),
			runtimeVersion.Str(),
		)

		// Update instrumentation libraries from all ScopeSpans, keyed by span kind
		for j := 0; j < resourceSpan.ScopeSpans().Len(); j++ {
			scopeSpan := resourceSpan.ScopeSpans().At(j)
			scope := scopeSpan.Scope()

			for k := 0; k < scopeSpan.Spans().Len(); k++ {
				span := scopeSpan.Spans().At(k)
				c.updateInstrumentationLibrary(svc, scope.Name(), scope.Version(), span.Kind().String(), span, attrs, now.UnixNano())
			}
		}
		c.servicesMu.Unlock()
	}

	return nil
}

// ConsumeMetrics method is called for each instance of metrics sent to the connector
func (c *serviceDiscoveryConnector) ConsumeMetrics(ctx context.Context, md pmetric.Metrics) error {
	// loop through the resource metrics
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		resourceMetric := md.ResourceMetrics().At(i)

		// Extract resource attributes
		attrs := resourceMetric.Resource().Attributes()
		serviceName, _ := attrs.Get("service.name")
		serviceNamespace, _ := attrs.Get("service.namespace")
		serviceVersion, _ := attrs.Get("service.version")
		deploymentEnvironment, _ := attrs.Get("deployment.environment")
		sdkLanguage, _ := attrs.Get("telemetry.sdk.language")
		sdkVersion, _ := attrs.Get("telemetry.sdk.version")
		runtimeName, _ := attrs.Get("process.runtime.name")
		runtimeVersion, _ := attrs.Get("process.runtime.version")

		c.logger.Debug("Processing metrics",
			zap.String("service.name", serviceName.Str()),
			zap.String("service.namespace", serviceNamespace.Str()),
			zap.String("deployment.environment", deploymentEnvironment.Str()),
			zap.Int("scope_metrics_count", resourceMetric.ScopeMetrics().Len()))

		now := time.Now()

		c.servicesMu.Lock()
		// Find or create the service
		svc := c.findOrCreateService(
			serviceName.Str(),
			serviceNamespace.Str(),
			serviceVersion.Str(),
			deploymentEnvironment.Str(),
			sdkLanguage.Str(),
			sdkVersion.Str(),
			runtimeName.Str(),
			runtimeVersion.Str(),
		)

		// Update instrumentation libraries from all ScopeMetrics.
		// Metrics have no spans, so span kind is left empty.
		for j := 0; j < resourceMetric.ScopeMetrics().Len(); j++ {
			scopeMetric := resourceMetric.ScopeMetrics().At(j)
			scope := scopeMetric.Scope()

			c.logger.Debug("Found instrumentation scope in metrics",
				zap.String("scope.name", scope.Name()),
				zap.String("scope.version", scope.Version()))

			c.updateInstrumentationLibrary(svc, scope.Name(), scope.Version(), "", ptrace.NewSpan(), pcommon.NewMap(), now.UnixNano())
		}
		c.servicesMu.Unlock()
	}

	return nil
}

// ConsumeLogs method is called for each instance of logs sent to the connector
func (c *serviceDiscoveryConnector) ConsumeLogs(ctx context.Context, ld plog.Logs) error {
	// loop through the resource logs
	for i := 0; i < ld.ResourceLogs().Len(); i++ {
		resourceLog := ld.ResourceLogs().At(i)

		// Extract resource attributes
		attrs := resourceLog.Resource().Attributes()
		serviceName, _ := attrs.Get("service.name")
		serviceNamespace, _ := attrs.Get("service.namespace")
		serviceVersion, _ := attrs.Get("service.version")
		deploymentEnvironment, _ := attrs.Get("deployment.environment")
		sdkLanguage, _ := attrs.Get("telemetry.sdk.language")
		sdkVersion, _ := attrs.Get("telemetry.sdk.version")
		runtimeName, _ := attrs.Get("process.runtime.name")
		runtimeVersion, _ := attrs.Get("process.runtime.version")

		now := time.Now()

		c.servicesMu.Lock()
		// Find or create the service
		svc := c.findOrCreateService(
			serviceName.Str(),
			serviceNamespace.Str(),
			serviceVersion.Str(),
			deploymentEnvironment.Str(),
			sdkLanguage.Str(),
			sdkVersion.Str(),
			runtimeName.Str(),
			runtimeVersion.Str(),
		)

		// Update instrumentation libraries from all ScopeLogs.
		// Logs have no spans, so span kind is left empty.
		for j := 0; j < resourceLog.ScopeLogs().Len(); j++ {
			scopeLog := resourceLog.ScopeLogs().At(j)
			scope := scopeLog.Scope()

			c.updateInstrumentationLibrary(svc, scope.Name(), scope.Version(), "", ptrace.NewSpan(), pcommon.NewMap(), now.UnixNano())
		}
		c.servicesMu.Unlock()
	}

	return nil
}

func (c *serviceDiscoveryConnector) evictExpiredEntries() {
	cutoff := time.Now().UnixNano() - int64(entryTTL)

	c.servicesMu.Lock()
	defer c.servicesMu.Unlock()

	surviving := c.opentelemetryServices[:0]
	for i := range c.opentelemetryServices {
		svc := &c.opentelemetryServices[i]
		libs := svc.InstrumentationLibraries[:0]
		for j := range svc.InstrumentationLibraries {
			if svc.InstrumentationLibraries[j].LastSeen >= cutoff {
				libs = append(libs, svc.InstrumentationLibraries[j])
			}
		}
		svc.InstrumentationLibraries = libs
		if len(libs) > 0 {
			surviving = append(surviving, *svc)
		}
	}
	c.opentelemetryServices = surviving
}

func (c *serviceDiscoveryConnector) logExportWorker() {
	defer c.wg.Done()

	for {
		select {
		case <-c.shutdownCh:
			return
		case <-c.logExportTicker.C:
			c.evictExpiredEntries()
			c.exportLogs()
		}
	}
}

// exportLogs exports one log record per service per instrumentation library,
// structured using the observe_transform format (matching the heartbeat receiver).
func (c *serviceDiscoveryConnector) exportLogs() {
	if c.logsConsumer == nil {
		return
	}

	ctx := context.Background()
	logs := plog.NewLogs()
	now := time.Now().UnixNano()
	validTo := now + int64(90*time.Minute)

	c.servicesMu.RLock()
	c.logger.Debug("Exporting service discovery logs", zap.Int("service_count", len(c.opentelemetryServices)))

	for _, svc := range c.opentelemetryServices {
		for _, il := range svc.InstrumentationLibraries {
			c.logger.Debug("Exporting service discovery",
				zap.String("service.name", svc.ServiceName),
				zap.String("service.namespace", svc.ServiceNamespace),
				zap.String("service.version", svc.ServiceVersion),
				zap.String("deployment.environment", svc.Environment),
				zap.String("library.name", il.Name),
				zap.String("library.version", il.Version),
				zap.String("span.kind", il.SpanKind))

			resourceLogs := logs.ResourceLogs().AppendEmpty()
			scopeLogs := resourceLogs.ScopeLogs().AppendEmpty()
			logRecord := scopeLogs.LogRecords().AppendEmpty()

			// -- observe_transform attribute --
			observeTransform := logRecord.Attributes().PutEmptyMap("observe_transform")

			// identifiers: the entity key
			identifiers := observeTransform.PutEmptyMap("identifiers")
			identifiers.PutStr("service.name", svc.ServiceName)
			identifiers.PutStr("service.namespace", svc.ServiceNamespace)
			identifiers.PutStr("deployment.environment.name", svc.Environment)
			identifiers.PutStr("instrumentation_library.name", il.Name)
			identifiers.PutStr("instrumentation_library.version", il.Version)
			if svc.ServiceVersion != "" {
				identifiers.PutStr("service.version", svc.ServiceVersion)
			}
			if il.SpanKind != "" {
				identifiers.PutStr("span.kind", il.SpanKind)
			}

			// facets: non-key metadata
			facets := observeTransform.PutEmptyMap("facets")
			facets.PutInt("last_seen", il.LastSeen)
			if il.HasSampleSpan {
				sampleMap := facets.PutEmptyMap("sample_span")
				sampleMap.PutStr("trace_id", il.SampleSpan.TraceID().String())
				sampleMap.PutStr("span_id", il.SampleSpan.SpanID().String())
				if !il.SampleSpan.ParentSpanID().IsEmpty() {
					sampleMap.PutStr("parent_span_id", il.SampleSpan.ParentSpanID().String())
				}
				sampleMap.PutStr("name", il.SampleSpan.Name())
				sampleMap.PutStr("kind", il.SampleSpan.Kind().String())
				sampleMap.PutInt("start_time_unix_nano", int64(il.SampleSpan.StartTimestamp()))
				sampleMap.PutInt("end_time_unix_nano", int64(il.SampleSpan.EndTimestamp()))
				sampleMap.PutStr("status.code", il.SampleSpan.Status().Code().String())
				if il.SampleSpan.Status().Message() != "" {
					sampleMap.PutStr("status.message", il.SampleSpan.Status().Message())
				}
				if il.SampleSpan.Attributes().Len() > 0 {
					attrMap := sampleMap.PutEmptyMap("attributes")
					il.SampleSpan.Attributes().CopyTo(attrMap)
				}
				if il.SampleResourceAttrs.Len() > 0 {
					resMap := sampleMap.PutEmptyMap("resource_attributes")
					il.SampleResourceAttrs.CopyTo(resMap)
				}
			}

			// validity window
			observeTransform.PutInt("valid_from", now)
			observeTransform.PutInt("valid_to", validTo)

			// kind and control
			observeTransform.PutStr("kind", "ServiceDiscovery")
			control := observeTransform.PutEmptyMap("control")
			control.PutStr("eventType", "HEARTBEAT")
			control.PutBool("isDelete", false)

		}
	}
	c.servicesMu.RUnlock()

	if logs.ResourceLogs().Len() > 0 {
		c.logsConsumer.ConsumeLogs(ctx, logs)
	}
}
