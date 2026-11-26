package observecol

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/apachereceiver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/consumer/consumertest"
	"go.opentelemetry.io/collector/receiver/receivertest"
)

// A minimal mod_status "?auto" page with the fields the receiver turns into metrics.
const apacheServerStatus = `ServerUptimeSeconds: 410
Total Accesses: 14169
Total kBytes: 20910
CPUUser: 0.12
CPUSystem: 0.08
CPUChildrenUser: 0
CPUChildrenSystem: 0
CPULoad: .000488
ReqPerSec: 34.56
BusyWorkers: 13
IdleWorkers: 227
ConnsTotal: 110
ConnsAsyncWriting: 3
ConnsAsyncKeepAlive: 5
ConnsAsyncClosing: 0
Load1: 0.9
Load5: 0.4
Load15: 0.3
Scoreboard: S_DD_L_GGG_____W__IIII_C________________W__________________________________.........................____WR______W____W________________________C______________________________________W_W____W______________R_________R________C_________WK_W________K_____W__C__________W___R______.............................................................................................................................
`

// TestApacheReceiverEmitsOnlyNewFormatMetrics checks that, with the agent's default feature
// gates, the apache receiver emits only the new metric names. The receiver is new to the agent,
// so no user has built anything on the original names and none will need migrating later.
func TestApacheReceiverEmitsOnlyNewFormatMetrics(t *testing.T) {
	ctx := context.Background()
	featureGates = []string{}
	t.Cleanup(func() { resetFeatureGates(t, ctx) })
	require.NoError(t, ApplyFeatureGates(ctx))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/server-status" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(apacheServerStatus))
	}))
	t.Cleanup(srv.Close)

	factory := apachereceiver.NewFactory()
	cfg := factory.CreateDefaultConfig().(*apachereceiver.Config)
	cfg.ClientConfig.Endpoint = srv.URL + "/server-status?auto"
	cfg.ControllerConfig.CollectionInterval = 50 * time.Millisecond

	sink := new(consumertest.MetricsSink)
	rcvr, err := factory.CreateMetrics(ctx, receivertest.NewNopSettings(factory.Type()), cfg, sink)
	require.NoError(t, err)
	require.NoError(t, rcvr.Start(ctx, componenttest.NewNopHost()))
	t.Cleanup(func() { require.NoError(t, rcvr.Shutdown(ctx)) })

	require.Eventually(t, func() bool { return len(sink.AllMetrics()) > 0 }, 10*time.Second, 50*time.Millisecond)

	names := map[string]bool{}
	md := sink.AllMetrics()[0]
	for i := 0; i < md.ResourceMetrics().Len(); i++ {
		sms := md.ResourceMetrics().At(i).ScopeMetrics()
		for j := 0; j < sms.Len(); j++ {
			ms := sms.At(j).Metrics()
			for k := 0; k < ms.Len(); k++ {
				names[ms.At(k).Name()] = true
			}
		}
	}

	for _, name := range []string{"apache.request.count", "apache.worker.status", "apache.worker.active", "apache.worker.idle"} {
		assert.True(t, names[name], "new-format metric %q should be emitted; got %v", name, names)
	}
	for _, name := range []string{"apache.requests", "apache.scoreboard", "apache.workers", "apache.current_connections"} {
		assert.False(t, names[name], "original-format metric %q should not be emitted; got %v", name, names)
	}
}
