package processdiscoveryreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func snapshot(pid int32, start uint64, runtime string) ProcessSnapshot {
	status := "unknown"
	if runtime != "" {
		status = "identified"
	}
	return ProcessSnapshot{
		ProcessIdentity:    ProcessIdentity{Key: ProcessKey{PID: pid, StartTime: start}},
		ProcessDescription: ProcessDescription{Command: runtime},
		RuntimeEvidence: RuntimeEvidence{
			RuntimeFamily: runtime, RuntimeStatus: status, RuntimeAssertions: []string{runtime + ".executable"},
		},
		ObservationProvenance: ObservationProvenance{Source: "procfs"},
	}
}

func TestProcessStateLifecycle(t *testing.T) {
	state := newProcessState()
	java := snapshot(10, 100, "java")
	events, _ := state.apply(ScanResult{Processes: []ProcessSnapshot{java}, Complete: true}, 2, 10, 0)
	require.Len(t, events, 1)
	assert.Equal(t, "process.started", events[0].Name)
	events, _ = state.apply(ScanResult{Processes: []ProcessSnapshot{java}, Complete: true}, 2, 10, 0)
	assert.Empty(t, events)
	changed := java
	changed.InferredVersion = "21"
	changed.InferredVersionKind = "release_metadata"
	events, _ = state.apply(ScanResult{Processes: []ProcessSnapshot{changed}, Complete: true}, 2, 10, 0)
	require.Len(t, events, 1)
	assert.Equal(t, "process.changed", events[0].Name)
	state.apply(ScanResult{Complete: true}, 2, 10, 0)
	events, _ = state.apply(ScanResult{Complete: true}, 2, 10, 0)
	require.Len(t, events, 1)
	assert.Equal(t, "process.stopped", events[0].Name)
}

func TestExplicitExit(t *testing.T) {
	state := newProcessState()
	state.applyObservation(snapshot(10, 100, "java"))
	events := state.applyExit(10, "ebpf_exit")
	require.Len(t, events, 1)
	assert.Equal(t, "ebpf_exit", events[0].StopReason)
	assert.Empty(t, state.tracked)
}

func TestUnmatchedExit(t *testing.T) {
	events := newProcessState().applyExit(10, "ebpf_exit")
	require.Len(t, events, 1)
	assert.Equal(t, "unavailable", events[0].Process.RuntimeStatus)
}

func TestIncompleteScanDoesNotStopProcess(t *testing.T) {
	state := newProcessState()
	state.apply(ScanResult{Processes: []ProcessSnapshot{snapshot(10, 100, "java")}, Complete: true}, 1, 10, 0)
	events, _ := state.apply(ScanResult{Complete: false}, 1, 10, 0)
	assert.Empty(t, events)
}

func TestProcessReappearanceResetsTerminationMisses(t *testing.T) {
	state := newProcessState()
	process := snapshot(10, 100, "java")
	state.apply(ScanResult{Processes: []ProcessSnapshot{process}, Complete: true}, 2, 10, 0)
	state.apply(ScanResult{Complete: true}, 2, 10, 0)
	state.apply(ScanResult{Processes: []ProcessSnapshot{process}, Complete: true}, 2, 10, 0)

	events, _ := state.apply(ScanResult{Complete: true}, 2, 10, 0)
	assert.Empty(t, events)
	events, _ = state.apply(ScanResult{Complete: true}, 2, 10, 0)
	require.Len(t, events, 1)
	assert.Equal(t, "process.stopped", events[0].Name)
}

func TestProcessLimitRetainsTrackedProcesses(t *testing.T) {
	state := newProcessState()
	tracked := snapshot(100, 100, "java")
	state.apply(ScanResult{Processes: []ProcessSnapshot{tracked}, Complete: true}, 1, 1, 0)

	events, saturated := state.apply(ScanResult{Processes: []ProcessSnapshot{
		snapshot(1, 100, "python"), tracked,
	}, Complete: true}, 1, 1, 0)
	assert.True(t, saturated)
	assert.Empty(t, events)
	require.Contains(t, state.tracked, tracked.Key)
}

func TestPIDReuseCreatesSeparateLifecycle(t *testing.T) {
	state := newProcessState()
	state.applyObservation(snapshot(10, 100, "java"))
	events := state.applyObservation(snapshot(10, 200, "java"))
	require.Len(t, events, 2)
	assert.Equal(t, "process.stopped", events[0].Name)
	assert.Equal(t, "process.started", events[1].Name)
}

func TestOldGenerationExitCannotDeleteReusedPID(t *testing.T) {
	state := newProcessState()
	state.applyObservation(snapshot(10, 100, "java"))
	state.applyObservation(snapshot(10, 200, "java"))
	events, _ := state.apply(ScanResult{Processes: []ProcessSnapshot{snapshot(10, 200, "java")}, Complete: true}, 1, 10, 0)
	assert.Empty(t, events)
	require.Len(t, state.tracked, 1)
	_, ok := state.tracked[ProcessKey{PID: 10, StartTime: 200}]
	assert.True(t, ok)
}

func TestProcessStateEmitsPeriodicSnapshot(t *testing.T) {
	state := newProcessState()
	process := snapshot(10, 100, "java")
	process.ObservedAt = time.Unix(100, 0)
	state.apply(ScanResult{Processes: []ProcessSnapshot{process}, Complete: true}, 1, 10, time.Minute)
	process.ObservedAt = time.Unix(161, 0)
	events, _ := state.apply(ScanResult{Processes: []ProcessSnapshot{process}, Complete: true}, 1, 10, time.Minute)
	require.Len(t, events, 1)
	assert.Equal(t, "process.snapshot", events[0].Name)
}
