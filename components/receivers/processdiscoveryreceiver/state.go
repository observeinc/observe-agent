package processdiscoveryreceiver

import (
	"cmp"
	"slices"
	"time"
)

type processEvent struct {
	Name          string
	Process       ProcessSnapshot
	ChangedFields []string
	StopReason    string
}

type trackedProcess struct {
	snapshot     ProcessSnapshot
	misses       int
	lastReported time.Time
}

type processState struct {
	tracked map[ProcessKey]trackedProcess
	byPID   map[int32]ProcessKey
}

func newProcessState() *processState {
	return &processState{tracked: make(map[ProcessKey]trackedProcess), byPID: make(map[int32]ProcessKey)}
}

func (s *processState) clone() *processState {
	cloned := newProcessState()
	for key, tracked := range s.tracked {
		cloned.tracked[key] = tracked
	}
	for pid, key := range s.byPID {
		cloned.byPID[pid] = key
	}
	return cloned
}

func (s *processState) applyObservation(snapshot ProcessSnapshot) []processEvent {
	if snapshot.Key.PID == 0 {
		return nil
	}
	if oldKey, ok := s.byPID[snapshot.Key.PID]; ok && oldKey != snapshot.Key {
		old := s.tracked[oldKey]
		delete(s.tracked, oldKey)
		delete(s.byPID, snapshot.Key.PID)
		events := []processEvent{{Name: "process.stopped", Process: old.snapshot, StopReason: "pid_reused"}}
		events = append(events, s.applyObservation(snapshot)...)
		return events
	}
	previous, ok := s.tracked[snapshot.Key]
	if !ok {
		s.tracked[snapshot.Key] = trackedProcess{snapshot: snapshot, lastReported: snapshot.ObservedAt}
		s.byPID[snapshot.Key.PID] = snapshot.Key
		return []processEvent{{Name: "process.started", Process: snapshot}}
	}
	changed := changedFields(previous.snapshot, snapshot)
	previous.snapshot = snapshot
	previous.misses = 0
	if len(changed) > 0 {
		previous.lastReported = snapshot.ObservedAt
	}
	s.tracked[snapshot.Key] = previous
	if len(changed) == 0 {
		return nil
	}
	return []processEvent{{Name: "process.changed", Process: snapshot, ChangedFields: changed}}
}

func (s *processState) applyExit(pid int32, reason string) []processEvent {
	key, ok := s.byPID[pid]
	if !ok {
		return []processEvent{{
			Name: "process.stopped",
			Process: ProcessSnapshot{
				ProcessIdentity:       ProcessIdentity{Key: ProcessKey{PID: pid}},
				RuntimeEvidence:       RuntimeEvidence{RuntimeStatus: "unavailable"},
				ObservationProvenance: ObservationProvenance{Source: "ebpf"},
			},
			StopReason: reason,
		}}
	}
	tracked := s.tracked[key]
	delete(s.byPID, pid)
	delete(s.tracked, key)
	return []processEvent{{Name: "process.stopped", Process: tracked.snapshot, StopReason: reason}}
}

func (s *processState) apply(scan ScanResult, graceScans, maxTracked int, reportInterval time.Duration) ([]processEvent, bool) {
	candidates := make(map[ProcessKey]ProcessSnapshot, len(scan.Processes))
	for _, snapshot := range scan.Processes {
		candidates[snapshot.Key] = snapshot
	}
	admitted := make(map[ProcessKey]ProcessSnapshot, min(len(candidates), maxTracked))
	for key := range s.tracked {
		if snapshot, ok := candidates[key]; ok && len(admitted) < maxTracked {
			admitted[key] = snapshot
		}
	}
	keys := make([]ProcessKey, 0, len(candidates))
	for key := range candidates {
		if _, ok := admitted[key]; !ok {
			keys = append(keys, key)
		}
	}
	slices.SortFunc(keys, func(left, right ProcessKey) int {
		if result := cmp.Compare(left.PID, right.PID); result != 0 {
			return result
		}
		return cmp.Compare(left.StartTime, right.StartTime)
	})
	for _, key := range keys {
		if len(admitted) >= maxTracked {
			break
		}
		admitted[key] = candidates[key]
	}

	var events []processEvent
	for _, snapshot := range admitted {
		observationEvents := s.applyObservation(snapshot)
		events = append(events, observationEvents...)
		if len(observationEvents) == 0 {
			tracked := s.tracked[snapshot.Key]
			if reportInterval > 0 && snapshot.ObservedAt.Sub(tracked.lastReported) >= reportInterval {
				tracked.lastReported = snapshot.ObservedAt
				s.tracked[snapshot.Key] = tracked
				events = append(events, processEvent{Name: "process.snapshot", Process: snapshot})
			}
		}
	}
	if scan.Complete {
		for key, tracked := range s.tracked {
			if _, ok := admitted[key]; ok {
				continue
			}
			if _, stillCandidate := candidates[key]; stillCandidate {
				continue
			}
			if _, unavailable := scan.UnavailablePIDs[key.PID]; unavailable {
				continue
			}
			tracked.misses++
			if tracked.misses < graceScans {
				s.tracked[key] = tracked
				continue
			}
			events = append(events, s.applyExit(key.PID, "snapshot_absent")...)
		}
	}
	slices.SortFunc(events, func(left, right processEvent) int {
		if result := cmp.Compare(left.Process.Key.PID, right.Process.Key.PID); result != 0 {
			return result
		}
		return cmp.Compare(eventOrder(left.Name), eventOrder(right.Name))
	})
	return events, len(candidates) > maxTracked
}

func eventOrder(name string) int {
	switch name {
	case "process.stopped":
		return 0
	case "process.started":
		return 1
	default:
		return 2
	}
}

func changedFields(before, after ProcessSnapshot) []string {
	changed := make([]string, 0, 2)
	if before.RuntimeName != after.RuntimeName || before.RuntimeFamily != after.RuntimeFamily || before.RuntimeVersion != after.RuntimeVersion || before.InferredVersion != after.InferredVersion || before.InferredVersionKind != after.InferredVersionKind || before.RuntimeStatus != after.RuntimeStatus || !slices.Equal(before.RuntimeAssertions, after.RuntimeAssertions) {
		changed = append(changed, "runtime")
	}
	if before.Command != after.Command || before.ExecutableName != after.ExecutableName || before.ParentPID != after.ParentPID || before.ExecutableArch != after.ExecutableArch {
		changed = append(changed, "process")
	}
	if before.ContainerID != after.ContainerID || before.K8sPodUID != after.K8sPodUID || before.K8sContainerName != after.K8sContainerName || before.K8sCorrelationStatus != after.K8sCorrelationStatus {
		changed = append(changed, "deployment")
	}
	return changed
}
