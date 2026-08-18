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
	cfg          *Config
	settings     receiver.Settings
	nextConsumer consumer.Logs
	emitter      processEmitter
	obsrecv      *receiverhelper.ObsReport
	telemetry    *receiverTelemetry
	source       ProcessSource
	lifecycle    LifecycleSource
	state        *processState
	supported    bool
	filter       *processFilter

	cancel          context.CancelFunc
	wg              sync.WaitGroup
	saturated       bool
	collectionMode  string
	eventLoss       uint64
	lifecycleEvents <-chan LifecycleEvent
	resourceAttrs   map[string]string
	exitReconcile   <-chan time.Time
	startOnce       sync.Once
	startErr        error
	references      atomic.Int32
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
		filter: newProcessFilter(cfg.Filtering),
		collectionMode: "snapshot_only", resourceAttrs: hostResourceAttributes(cfg),
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
			// Process exited before we could inspect it. The next periodic
			// scan will reconcile state; avoid expensive full scans here.
			return
		}
		snapshot.Source, snapshot.ObservedAt, snapshot.CgroupID = "ebpf", event.ObservedAt, event.CgroupID
		if !r.filter.ShouldInclude(snapshot) {
			return
		}
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
	}
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
	events, saturated := nextState.apply(scan, r.cfg.TerminationGraceScans, r.cfg.MaxTrackedProcesses, r.cfg.Filtering.MinLifetimeScans, r.cfg.ReportInterval)
	r.saturated = saturated

	// Drain buffered lifecycle events that arrived during the scan, but
	// enforce a deadline so we don't spin indefinitely when eBPF events
	// arrive faster than we can process them.
	deadline := time.After(2 * time.Second)
	draining := r.lifecycleEvents != nil
	for draining {
		select {
		case event, ok := <-r.lifecycleEvents:
			if !ok {
				r.lifecycleEvents = nil
				r.collectionMode = "snapshot_only"
				draining = false
				continue
			}
			switch event.Type {
			case lifecycleLoss:
				r.eventLoss += event.Lost
			case lifecycleExec:
				snapshot, inspectErr := r.source.Inspect(ctx, event.PID)
				if inspectErr == nil {
					snapshot.Source = "ebpf"
					snapshot.ObservedAt = event.ObservedAt
					snapshot.CgroupID = event.CgroupID
					if !r.filter.ShouldInclude(snapshot) {
						continue
					}
					observedEvents := nextState.applyObservation(snapshot)
					for index := range observedEvents {
						if observedEvents[index].Name == "process.changed" {
							observedEvents[index].Name = "process.exec"
						}
					}
					events = append(events, observedEvents...)
				}
			case lifecycleExit:
				// Exit events during initial drain are handled by the
				// snapshot scan that already ran; skip expensive re-scans.
			}
		case <-deadline:
			draining = false
		default:
			draining = false
		}
	}
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
	events, saturated := nextState.apply(scan, r.cfg.TerminationGraceScans, r.cfg.MaxTrackedProcesses, r.cfg.Filtering.MinLifetimeScans, r.cfg.ReportInterval)
	if saturated && !r.saturated {
		r.settings.Logger.Warn("process discovery tracked process limit reached", zap.Int("limit", r.cfg.MaxTrackedProcesses), zap.Int("candidates", len(scan.Processes)))
		r.telemetry.saturation.Add(ctx, 1)
	}
	r.saturated = saturated
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
