package processdiscoveryreceiver

import (
	"context"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type receiverTelemetry struct {
	scans                 metric.Int64Counter
	scanDuration          metric.Float64Histogram
	processesObserved     metric.Int64Histogram
	processesInaccessible metric.Int64Counter
	lifecycleEvents       metric.Int64Counter
	lifecycleEventsLost   metric.Int64Counter
	recordsEmitted        metric.Int64Counter
	saturation            metric.Int64Counter
	networkSeriesDropped  metric.Int64Counter
}

func newReceiverTelemetry(settings component.TelemetrySettings) (*receiverTelemetry, error) {
	meter := settings.MeterProvider.Meter("github.com/observeinc/observe-agent/components/receivers/processdiscoveryreceiver")
	telemetry := &receiverTelemetry{}
	var err error
	if telemetry.scans, err = meter.Int64Counter("otelcol_process_discovery_scans", metric.WithUnit("{scans}")); err != nil {
		return nil, err
	}
	if telemetry.scanDuration, err = meter.Float64Histogram("otelcol_process_discovery_scan_duration", metric.WithUnit("s")); err != nil {
		return nil, err
	}
	if telemetry.processesObserved, err = meter.Int64Histogram("otelcol_process_discovery_processes_observed", metric.WithUnit("{processes}")); err != nil {
		return nil, err
	}
	if telemetry.processesInaccessible, err = meter.Int64Counter("otelcol_process_discovery_processes_inaccessible", metric.WithUnit("{processes}")); err != nil {
		return nil, err
	}
	if telemetry.lifecycleEvents, err = meter.Int64Counter("otelcol_process_discovery_lifecycle_events", metric.WithUnit("{events}")); err != nil {
		return nil, err
	}
	if telemetry.lifecycleEventsLost, err = meter.Int64Counter("otelcol_process_discovery_lifecycle_events_lost", metric.WithUnit("{events}")); err != nil {
		return nil, err
	}
	if telemetry.recordsEmitted, err = meter.Int64Counter("otelcol_process_discovery_records_emitted", metric.WithUnit("{records}")); err != nil {
		return nil, err
	}
	if telemetry.saturation, err = meter.Int64Counter("otelcol_process_discovery_saturation", metric.WithUnit("{scans}")); err != nil {
		return nil, err
	}
	if telemetry.networkSeriesDropped, err = meter.Int64Counter("otelcol_process_discovery_network_series_dropped", metric.WithUnit("{series}")); err != nil {
		return nil, err
	}
	return telemetry, nil
}

func (t *receiverTelemetry) recordScan(ctx context.Context, started time.Time, scan ScanResult, err error) {
	status := "complete"
	if err != nil {
		status = "failed"
	} else if !scan.Complete {
		status = "partial"
	}
	t.scans.Add(ctx, 1, metric.WithAttributes(attribute.String("status", status)))
	t.scanDuration.Record(ctx, time.Since(started).Seconds())
	t.processesObserved.Record(ctx, int64(len(scan.Processes)))
	if count := len(scan.UnavailablePIDs); count > 0 {
		t.processesInaccessible.Add(ctx, int64(count))
	}
}

func (t *receiverTelemetry) recordLifecycle(ctx context.Context, event LifecycleEvent) {
	t.lifecycleEvents.Add(ctx, 1, metric.WithAttributes(attribute.String("event_type", string(event.Type))))
	if event.Type == lifecycleLoss && event.Lost > 0 {
		t.lifecycleEventsLost.Add(ctx, int64(event.Lost))
	}
}

func (t *receiverTelemetry) recordEmitted(ctx context.Context, events []processEvent) {
	for _, event := range events {
		t.recordsEmitted.Add(ctx, 1, metric.WithAttributes(attribute.String("event_type", eventName(event.Name))))
	}
}
