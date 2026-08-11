package processdiscoveryreceiver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/receiverhelper"
	"go.uber.org/zap"
)

type processDiscoveryReceiver struct {
	cfg                 *Config
	settings            receiver.Settings
	nextConsumer        consumer.Logs
	nextMetricsConsumer consumer.Metrics
	emitter             processEmitter
	obsrecv             *receiverhelper.ObsReport
	telemetry           *receiverTelemetry
	source              ProcessSource
	lifecycle           LifecycleSource
	state               *processState
	supported           bool

	cancel           context.CancelFunc
	wg               sync.WaitGroup
	saturated        bool
	collectionMode   string
	eventLoss        uint64
	lifecycleEvents  <-chan LifecycleEvent
	resourceAttrs    map[string]string
	exitReconcile    <-chan time.Time
	startOnce        sync.Once
	startErr         error
	references       atomic.Int32
	networkMetrics   *networkMetricState
	acceptingSockets map[int32]NetworkEndpoint
}

const exitReconcileDelay = 100 * time.Millisecond

func newReceiver(set receiver.Settings, cfg *Config) (*processDiscoveryReceiver, error) {
	source, err := newProcessSource(cfg)
	if err != nil {
		return nil, err
	}
	lifecycle, err := newLifecycleSource(cfg, set.Logger)
	if err != nil {
		return nil, err
	}
	obsrecv, err := receiverhelper.NewObsReport(receiverhelper.ObsReportSettings{
		LongLivedCtx: true, ReceiverID: set.ID, ReceiverCreateSettings: set,
	})
	if err != nil {
		return nil, err
	}
	telemetry, err := newReceiverTelemetry(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}
	receiver := &processDiscoveryReceiver{
		cfg: cfg, settings: set, obsrecv: obsrecv, telemetry: telemetry,
		source: source, lifecycle: lifecycle, state: newProcessState(), supported: platformSupported(),
		collectionMode: "snapshot_only", resourceAttrs: hostResourceAttributes(cfg),
		networkMetrics:   newNetworkMetricState(cfg.Network.MaxSeries),
		acceptingSockets: make(map[int32]NetworkEndpoint),
	}
	receiver.emitter = newLogsEmitter(receiver)
	return receiver, nil
}

func (r *processDiscoveryReceiver) addReference() *processDiscoveryReceiver {
	r.references.Add(1)
	return r
}

func hostResourceAttributes(cfg *Config) map[string]string {
	attributes := map[string]string{"os.type": runtime.GOOS}
	if cfg.HostArch != "" {
		attributes["host.arch"] = cfg.HostArch
	} else if filepath.Clean(cfg.ProcFSPath) == defaultProcFSPath {
		attributes["host.arch"] = runtime.GOARCH
	}
	if cfg.HostID != "" {
		attributes["host.id"] = cfg.HostID
		return attributes
	}
	machineIDPath := cfg.HostMachineIDPath
	if machineIDPath == "" && filepath.Clean(cfg.ProcFSPath) == defaultProcFSPath {
		machineIDPath = "/etc/machine-id"
	}
	if machineIDPath == "" {
		return attributes
	}
	if data, err := os.ReadFile(machineIDPath); err == nil {
		if hostID := strings.TrimSpace(string(data)); hostID != "" {
			attributes["host.id"] = hostID
		}
	}
	return attributes
}

func (r *processDiscoveryReceiver) Start(_ context.Context, _ component.Host) error {
	r.startOnce.Do(func() {
		if !r.supported {
			r.startErr = fmt.Errorf("processdiscovery receiver is not supported on this platform")
			return
		}
		workerCtx, cancel := context.WithCancel(context.Background())
		r.cancel = cancel
		events, err := r.lifecycle.Start(workerCtx)
		if err != nil {
			r.settings.Logger.Warn("process lifecycle source unavailable; using snapshots", zap.Error(err))
		} else if events != nil {
			r.collectionMode = "snapshot_and_events"
			r.lifecycleEvents = events
		}
		r.wg.Add(1)
		go r.run(workerCtx)
	})
	return r.startErr
}

func (r *processDiscoveryReceiver) Shutdown(context.Context) error {
	if r.references.Add(-1) > 0 {
		return nil
	}
	if r.cancel != nil {
		r.cancel()
	}
	err := r.lifecycle.Close()
	r.wg.Wait()
	sharedReceivers.Lock()
	delete(sharedReceivers.items, r.settings.ID)
	sharedReceivers.Unlock()
	return err
}

func (r *processDiscoveryReceiver) run(ctx context.Context) {
	defer r.wg.Done()
	events := r.lifecycleEvents
	if !waitFor(ctx, r.cfg.InitialDelay) {
		return
	}
	r.initialScanAndEmit(ctx)
	ticker := time.NewTicker(r.cfg.CollectionInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.scanAndEmit(ctx)
		case <-r.exitReconcile:
			r.exitReconcile = nil
			r.scanAndEmit(ctx)
		case event, ok := <-events:
			if !ok {
				events = nil
				r.collectionMode = "snapshot_only"
				continue
			}
			r.applyLifecycleEvent(ctx, event)
		}
	}
}

func (r *processDiscoveryReceiver) applyLifecycleEvent(ctx context.Context, event LifecycleEvent) {
	r.telemetry.recordLifecycle(ctx, event)
	switch event.Type {
	case lifecycleLoss:
		r.eventLoss += event.Lost
		r.scanAndEmit(ctx)
	case lifecycleExec:
		snapshot, err := r.source.Inspect(ctx, event.PID)
		if err != nil {
			// Without generation evidence an unavailable exec cannot safely mutate
			// state for a PID that may already have been reused.
			r.scanAndEmit(ctx)
			return
		}
		snapshot.Source, snapshot.ObservedAt, snapshot.CgroupID = "ebpf", event.ObservedAt, event.CgroupID
		r.applyAndEmit(ctx, func(state *processState) []processEvent {
			events := state.applyObservation(snapshot)
			for index := range events {
				if events[index].Name == "process.changed" {
					events[index].Name = "process.exec"
				}
			}
			return events
		})
	case lifecycleExit:
		// Tracepoint exits do not carry process generation. Reconcile rather than
		// deleting whichever process currently owns a reused PID. A short delay
		// coalesces exit bursts while keeping reconciliation independent of whether
		// the exiting PID remains inspectable or has already been reused.
		if r.exitReconcile == nil {
			r.exitReconcile = time.After(exitReconcileDelay)
		}
	case lifecycleAccept:
		r.recordAcceptedConnection(event)
	case lifecycleAcceptEnter:
		r.recordAcceptEntry(event)
	}
}

func (r *processDiscoveryReceiver) recordAcceptEntry(event LifecycleEvent) {
	resolver, ok := r.source.(AcceptedConnectionResolver)
	if !ok {
		return
	}
	endpoint, err := resolver.ResolveAcceptedConnection(event.PID, event.FD)
	if err == nil {
		r.acceptingSockets[event.ThreadID] = endpoint
	}
}

func (r *processDiscoveryReceiver) recordAcceptedConnection(event LifecycleEvent) {
	key, ok := r.state.byPID[event.PID]
	if !ok {
		return
	}
	endpoint, ok := r.acceptingSockets[event.ThreadID]
	delete(r.acceptingSockets, event.ThreadID)
	if !ok {
		return
	}
	r.networkMetrics.recordAccepted(r.state.tracked[key].snapshot, endpoint, event.ObservedAt)
}

func (r *processDiscoveryReceiver) initialScanAndEmit(ctx context.Context) {
	started := time.Now()
	scan, err := r.source.Scan(ctx)
	r.telemetry.recordScan(ctx, started, scan, err)
	if err != nil && ctx.Err() == nil {
		r.settings.Logger.Warn("initial process discovery scan failed", zap.Error(err))
		return
	}
	nextState := r.state.clone()
	events, saturated := nextState.apply(scan, r.cfg.TerminationGraceScans, r.cfg.MaxTrackedProcesses, r.cfg.ReportInterval)
	r.saturated = saturated
	for r.lifecycleEvents != nil {
		select {
		case event, ok := <-r.lifecycleEvents:
			if !ok {
				r.lifecycleEvents = nil
				r.collectionMode = "snapshot_only"
				continue
			}
			switch event.Type {
			case lifecycleLoss:
				r.eventLoss += event.Lost
				scan, _ = r.source.Scan(ctx)
				nextState = r.state.clone()
				events, saturated = nextState.apply(scan, r.cfg.TerminationGraceScans, r.cfg.MaxTrackedProcesses, r.cfg.ReportInterval)
				r.saturated = saturated
			case lifecycleExec:
				snapshot, inspectErr := r.source.Inspect(ctx, event.PID)
				if inspectErr == nil {
					snapshot.Source = "ebpf"
					snapshot.ObservedAt = event.ObservedAt
					snapshot.CgroupID = event.CgroupID
					observedEvents := nextState.applyObservation(snapshot)
					for index := range observedEvents {
						if observedEvents[index].Name == "process.changed" {
							observedEvents[index].Name = "process.exec"
						}
					}
					events = append(events, observedEvents...)
				} else {
					scan, _ = r.source.Scan(ctx)
					nextState = r.state.clone()
					events, saturated = nextState.apply(scan, r.cfg.TerminationGraceScans, r.cfg.MaxTrackedProcesses, r.cfg.ReportInterval)
					r.saturated = saturated
				}
			case lifecycleExit:
				scan, _ = r.source.Scan(ctx)
				nextState = r.state.clone()
				events, saturated = nextState.apply(scan, r.cfg.TerminationGraceScans, r.cfg.MaxTrackedProcesses, r.cfg.ReportInterval)
				r.saturated = saturated
			case lifecycleAcceptEnter:
				r.recordAcceptEntry(event)
			case lifecycleAccept:
				delete(r.acceptingSockets, event.ThreadID)
			}
		default:
			r.emitMetrics(ctx, scan, time.Now())
			r.emitAndCommit(ctx, nextState, events)
			return
		}
	}
	r.emitMetrics(ctx, scan, time.Now())
	r.emitAndCommit(ctx, nextState, events)
}

func (r *processDiscoveryReceiver) scanAndEmit(ctx context.Context) {
	started := time.Now()
	scan, err := r.source.Scan(ctx)
	r.telemetry.recordScan(ctx, started, scan, err)
	if err != nil && ctx.Err() == nil {
		r.settings.Logger.Warn("process discovery scan failed", zap.Error(err))
		return
	}
	nextState := r.state.clone()
	events, saturated := nextState.apply(scan, r.cfg.TerminationGraceScans, r.cfg.MaxTrackedProcesses, r.cfg.ReportInterval)
	if saturated && !r.saturated {
		r.settings.Logger.Warn("process discovery tracked process limit reached", zap.Int("limit", r.cfg.MaxTrackedProcesses), zap.Int("candidates", len(scan.Processes)))
		r.telemetry.saturation.Add(ctx, 1)
	}
	r.saturated = saturated
	for key := range r.networkMetrics.accepted {
		if _, ok := nextState.tracked[key.Process]; !ok {
			r.networkMetrics.removeProcess(key.Process)
		}
	}
	r.emitMetrics(ctx, scan, time.Now())
	r.emitAndCommit(ctx, nextState, events)
}

func (r *processDiscoveryReceiver) applyAndEmit(ctx context.Context, apply func(*processState) []processEvent) {
	nextState := r.state.clone()
	r.emitAndCommit(ctx, nextState, apply(nextState))
}

func (r *processDiscoveryReceiver) emitAndCommit(ctx context.Context, nextState *processState, events []processEvent) {
	if len(events) == 0 {
		r.state = nextState
		return
	}
	err := r.emitter.Emit(ctx, events)
	if err != nil {
		r.settings.Logger.Warn("failed to emit process discovery events", zap.Error(err))
		return
	}
	r.telemetry.recordEmitted(ctx, events)
	r.state = nextState
}

func waitFor(ctx context.Context, delay time.Duration) bool {
	if delay == 0 {
		return true
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func stableIdentity(key ProcessKey) string {
	if !key.CreationTime.IsZero() {
		return fmt.Sprintf("%d:%s", key.PID, key.CreationTime.UTC().Format(time.RFC3339Nano))
	}
	return fmt.Sprintf("%d:%d", key.PID, key.StartTime)
}
