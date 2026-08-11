package processdiscoveryreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pmetric"
)

func TestBuildMetricsSeparatesEntityAndPortDimensions(t *testing.T) {
	process := snapshot(10, 100, "java")
	process.Key.CreationTime = time.Unix(10, 0)
	process.Endpoints = []NetworkEndpoint{{LocalPort: 8080, Type: "ipv4", State: "listen", Count: 1}}
	key := networkSeriesKey{Process: process.Key, LocalPort: 8080, NetworkType: "ipv4"}
	metrics, dropped := buildMetrics([]ProcessSnapshot{process}, map[networkSeriesKey]acceptedSeries{
		key: {Count: 3, StartTime: time.Unix(20, 0)},
	}, map[string]string{"host.id": "host-1"}, time.Unix(30, 0), 10)

	assert.Zero(t, dropped)
	require.Equal(t, 2, metrics.DataPointCount())
	resource := metrics.ResourceMetrics().At(0).Resource().Attributes()
	pid, ok := resource.Get("process.pid")
	require.True(t, ok)
	assert.Equal(t, int64(10), pid.Int())
	assert.Equal(t, "host-1", resource.AsRaw()["host.id"])

	accepted := findMetric(t, metrics, acceptedConnectionsMetric)
	assert.Equal(t, pmetric.MetricTypeSum, accepted.Type())
	assert.True(t, accepted.Sum().IsMonotonic())
	assert.Equal(t, pmetric.AggregationTemporalityCumulative, accepted.Sum().AggregationTemporality())
	point := accepted.Sum().DataPoints().At(0)
	assert.Equal(t, int64(3), point.IntValue())
	assert.Equal(t, int64(8080), point.Attributes().AsRaw()["network.local.port"])
}

func TestAcceptedSeriesAreBoundedAndGenerationSafe(t *testing.T) {
	state := newNetworkMetricState(1)
	first := snapshot(10, 100, "java")
	second := snapshot(10, 200, "java")
	state.recordAccepted(first, NetworkEndpoint{LocalPort: 80, Type: "ipv4"}, time.Unix(1, 0))
	state.recordAccepted(second, NetworkEndpoint{LocalPort: 80, Type: "ipv4"}, time.Unix(2, 0))
	require.Len(t, state.accepted, 1)
	assert.Equal(t, uint64(1), state.dropped)
	state.removeProcess(first.Key)
	assert.Empty(t, state.accepted)
}

func TestBuildMetricsBoundsPublicSeries(t *testing.T) {
	process := snapshot(10, 100, "java")
	process.Endpoints = []NetworkEndpoint{
		{LocalPort: 80, Type: "ipv4", State: "listen", Count: 1},
		{LocalPort: 443, Type: "ipv4", State: "listen", Count: 1},
	}
	metrics, dropped := buildMetrics([]ProcessSnapshot{process}, nil, nil, time.Unix(30, 0), 1)
	assert.Equal(t, uint64(1), dropped)
	assert.Equal(t, 1, metrics.DataPointCount())
}

func TestStaleExitDoesNotRemoveReusedPIDSeries(t *testing.T) {
	r := newTestReceiver(t, nil)
	newGeneration := snapshot(10, 200, "java")
	r.state.applyObservation(newGeneration)
	r.networkMetrics.recordAccepted(newGeneration, NetworkEndpoint{LocalPort: 8080, Type: "ipv4"}, time.Unix(2, 0))

	r.applyLifecycleEvent(t.Context(), LifecycleEvent{Type: lifecycleExit, PID: 10})
	require.Len(t, r.networkMetrics.accepted, 1)
}

func TestAcceptEntryAndExitIncrementCounterWithoutUsingReturnedFD(t *testing.T) {
	r := newTestReceiver(t, nil)
	process := snapshot(10, 100, "java")
	r.state.applyObservation(process)
	r.acceptingSockets[11] = NetworkEndpoint{LocalPort: 8080, Type: "ipv4"}

	r.applyLifecycleEvent(t.Context(), LifecycleEvent{Type: lifecycleAccept, PID: 10, ThreadID: 11, FD: 99, ObservedAt: time.Unix(2, 0)})
	require.Len(t, r.networkMetrics.accepted, 1)
	assert.Empty(t, r.acceptingSockets)
}

func findMetric(t *testing.T, metrics pmetric.Metrics, name string) pmetric.Metric {
	t.Helper()
	for i := 0; i < metrics.ResourceMetrics().Len(); i++ {
		for j := 0; j < metrics.ResourceMetrics().At(i).ScopeMetrics().Len(); j++ {
			items := metrics.ResourceMetrics().At(i).ScopeMetrics().At(j).Metrics()
			for k := 0; k < items.Len(); k++ {
				if items.At(k).Name() == name {
					return items.At(k)
				}
			}
		}
	}
	require.FailNow(t, "metric not found", name)
	return pmetric.NewMetric()
}
