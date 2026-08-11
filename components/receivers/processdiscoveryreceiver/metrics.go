package processdiscoveryreceiver

import (
	"context"
	"sort"
	"strconv"
	"time"

	"github.com/observeinc/observe-agent/components/receivers/processdiscoveryreceiver/internal/metadata"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

const (
	acceptedConnectionsMetric = "process.network.connection.accepted"
	connectionCountMetric     = "process.network.connection.count"
)

type networkSeriesKey struct {
	Process     ProcessKey
	LocalPort   int
	NetworkType string
}

type acceptedSeries struct {
	Count     int64
	StartTime time.Time
}

type networkMetricState struct {
	accepted map[networkSeriesKey]acceptedSeries
	dropped  uint64
	max      int
}

func newNetworkMetricState(maxSeries int) *networkMetricState {
	return &networkMetricState{accepted: make(map[networkSeriesKey]acceptedSeries), max: maxSeries}
}

func (s *networkMetricState) recordAccepted(process ProcessSnapshot, endpoint NetworkEndpoint, observedAt time.Time) {
	key := networkSeriesKey{Process: process.Key, LocalPort: endpoint.LocalPort, NetworkType: endpoint.Type}
	series, ok := s.accepted[key]
	if !ok {
		if len(s.accepted) >= s.max {
			s.dropped++
			return
		}
		series.StartTime = observedAt
		if !process.Key.CreationTime.IsZero() && process.Key.CreationTime.After(series.StartTime) {
			series.StartTime = process.Key.CreationTime
		}
	}
	series.Count++
	s.accepted[key] = series
}

func (s *networkMetricState) removeProcess(key ProcessKey) {
	for seriesKey := range s.accepted {
		if seriesKey.Process == key {
			delete(s.accepted, seriesKey)
		}
	}
}

func buildMetrics(processes []ProcessSnapshot, accepted map[networkSeriesKey]acceptedSeries, resourceAttributes map[string]string, observedAt time.Time, maxSeries int) (pmetric.Metrics, uint64) {
	metrics := pmetric.NewMetrics()
	var emittedSeries int
	var dropped uint64
	processByKey := make(map[ProcessKey]ProcessSnapshot, len(processes))
	for _, process := range processes {
		processByKey[process.Key] = process
	}
	resources := make(map[ProcessKey]pmetric.ResourceMetrics)
	metricByProcess := make(map[ProcessKey]map[string]pmetric.Metric)
	resourceFor := func(process ProcessSnapshot) pmetric.ResourceMetrics {
		if existing, ok := resources[process.Key]; ok {
			return existing
		}
		resourceMetrics := metrics.ResourceMetrics().AppendEmpty()
		putMetricResource(resourceMetrics.Resource().Attributes(), process, resourceAttributes)
		resourceMetrics.ScopeMetrics().AppendEmpty().Scope().SetName(metadata.ScopeName)
		resources[process.Key] = resourceMetrics
		return resourceMetrics
	}
	metricFor := func(process ProcessSnapshot, name, description string, sum bool) pmetric.Metric {
		byName := metricByProcess[process.Key]
		if byName == nil {
			byName = make(map[string]pmetric.Metric)
			metricByProcess[process.Key] = byName
		}
		if existing, ok := byName[name]; ok {
			return existing
		}
		var metric pmetric.Metric
		if sum {
			metric = appendCumulativeSum(resourceFor(process), name, description)
		} else {
			metric = appendGauge(resourceFor(process), name, description)
		}
		byName[name] = metric
		return metric
	}

	for _, process := range processes {
		if len(process.Endpoints) == 0 {
			continue
		}
		metric := metricFor(process, connectionCountMetric, "Current number of process network connections by state.", false)
		for _, endpoint := range process.Endpoints {
			if emittedSeries >= maxSeries {
				dropped++
				continue
			}
			point := metric.Gauge().DataPoints().AppendEmpty()
			point.SetTimestamp(pcommon.NewTimestampFromTime(observedAt))
			point.SetIntValue(endpoint.Count)
			putNetworkAttributes(point.Attributes(), endpoint)
			emittedSeries++
		}
	}
	acceptedKeys := make([]networkSeriesKey, 0, len(accepted))
	for key := range accepted {
		acceptedKeys = append(acceptedKeys, key)
	}
	sort.Slice(acceptedKeys, func(i, j int) bool {
		left, right := acceptedKeys[i], acceptedKeys[j]
		if left.Process.PID != right.Process.PID {
			return left.Process.PID < right.Process.PID
		}
		if left.Process.StartTime != right.Process.StartTime {
			return left.Process.StartTime < right.Process.StartTime
		}
		if left.LocalPort != right.LocalPort {
			return left.LocalPort < right.LocalPort
		}
		return left.NetworkType < right.NetworkType
	})
	for _, key := range acceptedKeys {
		series := accepted[key]
		process, ok := processByKey[key.Process]
		if !ok {
			continue
		}
		if emittedSeries >= maxSeries {
			dropped++
			continue
		}
		metric := metricFor(process, acceptedConnectionsMetric, "Total number of TCP connections accepted by the process.", true)
		point := metric.Sum().DataPoints().AppendEmpty()
		point.SetStartTimestamp(pcommon.NewTimestampFromTime(series.StartTime))
		point.SetTimestamp(pcommon.NewTimestampFromTime(observedAt))
		point.SetIntValue(series.Count)
		putNetworkAttributes(point.Attributes(), NetworkEndpoint{LocalPort: key.LocalPort, Type: key.NetworkType})
		emittedSeries++
	}
	return metrics, dropped
}

func appendGauge(resourceMetrics pmetric.ResourceMetrics, name, description string) pmetric.Metric {
	scopeMetrics := resourceMetrics.ScopeMetrics().At(0)
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(description)
	metric.SetUnit("{connection}")
	metric.SetEmptyGauge()
	return metric
}

func appendCumulativeSum(resourceMetrics pmetric.ResourceMetrics, name, description string) pmetric.Metric {
	scopeMetrics := resourceMetrics.ScopeMetrics().At(0)
	metric := scopeMetrics.Metrics().AppendEmpty()
	metric.SetName(name)
	metric.SetDescription(description)
	metric.SetUnit("{connection}")
	sum := metric.SetEmptySum()
	sum.SetIsMonotonic(true)
	sum.SetAggregationTemporality(pmetric.AggregationTemporalityCumulative)
	return metric
}

func putMetricResource(attrs pcommon.Map, process ProcessSnapshot, resourceAttributes map[string]string) {
	for key, value := range resourceAttributes {
		attrs.PutStr(key, value)
	}
	attrs.PutInt("process.pid", int64(process.Key.PID))
	if !process.Key.CreationTime.IsZero() {
		attrs.PutStr("process.creation.time", process.Key.CreationTime.UTC().Format(time.RFC3339Nano))
	} else {
		attrs.PutStr("process.discovery.generation", strconv.FormatUint(process.Key.StartTime, 10))
	}
	putString(attrs, "process.command", process.Command)
	putString(attrs, "process.executable.name", process.ExecutableName)
	putString(attrs, "process.runtime.name", process.RuntimeName)
	putString(attrs, "process.discovery.runtime.family", process.RuntimeFamily)
	putString(attrs, "container.id", process.ContainerID)
	putString(attrs, "k8s.pod.uid", process.K8sPodUID)
	putString(attrs, "k8s.namespace.name", process.K8sNamespaceName)
}

func putNetworkAttributes(attrs pcommon.Map, endpoint NetworkEndpoint) {
	attrs.PutStr("network.transport", "tcp")
	attrs.PutInt("network.local.port", int64(endpoint.LocalPort))
	putString(attrs, "network.type", endpoint.Type)
	putString(attrs, "network.connection.state", endpoint.State)
}

func (r *processDiscoveryReceiver) emitMetrics(ctx context.Context, scan ScanResult, observedAt time.Time) {
	if r.nextMetricsConsumer == nil {
		return
	}
	metrics, dropped := buildMetrics(scan.Processes, r.networkMetrics.accepted, r.resourceAttrs, observedAt, r.cfg.Network.MaxSeries)
	r.networkMetrics.dropped += dropped
	if metrics.DataPointCount() == 0 {
		return
	}
	obsCtx := r.obsrecv.StartMetricsOp(ctx)
	err := r.nextMetricsConsumer.ConsumeMetrics(obsCtx, metrics)
	r.obsrecv.EndMetricsOp(obsCtx, metadata.Type.String(), metrics.DataPointCount(), err)
	if r.networkMetrics.dropped > 0 {
		r.telemetry.networkSeriesDropped.Add(ctx, int64(r.networkMetrics.dropped))
		r.networkMetrics.dropped = 0
	}
}
