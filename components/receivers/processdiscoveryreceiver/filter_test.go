package processdiscoveryreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPreFilterBlocksSystemDenylist(t *testing.T) {
	f := newProcessFilter(FilterConfig{ExcludeSystemProcesses: true, MinLifetimeScans: 2})
	for _, name := range []string{"bash", "sh", "zsh", "sleep", "pause", "sudo", "make", "gcc"} {
		assert.True(t, f.PreFilter(name), "expected %q to be pre-filtered", name)
	}
}

func TestPreFilterBlocksDenylistPrefixes(t *testing.T) {
	f := newProcessFilter(FilterConfig{ExcludeSystemProcesses: true, MinLifetimeScans: 2})
	assert.True(t, f.PreFilter("containerd-shim"))
	assert.True(t, f.PreFilter("containerd-shim-runc-v2"))
	assert.True(t, f.PreFilter("systemd-logind"))
	assert.True(t, f.PreFilter("systemd-journald"))
}

func TestPreFilterAllowsApplications(t *testing.T) {
	f := newProcessFilter(FilterConfig{ExcludeSystemProcesses: true, MinLifetimeScans: 2})
	for _, name := range []string{"java", "python3", "node", "apiserver", "nginx", "postgres"} {
		assert.False(t, f.PreFilter(name), "expected %q to pass pre-filter", name)
	}
}

func TestPreFilterDisabledPassesEverything(t *testing.T) {
	f := newProcessFilter(FilterConfig{ExcludeSystemProcesses: false, MinLifetimeScans: 2})
	assert.False(t, f.PreFilter("bash"))
	assert.False(t, f.PreFilter("sleep"))
	assert.False(t, f.PreFilter("pause"))
}

func TestPreFilterUserIncludeOverridesDenylist(t *testing.T) {
	f := newProcessFilter(FilterConfig{
		ExcludeSystemProcesses: true,
		MinLifetimeScans:       2,
		Include:                MatchConfig{ExecutableNames: []string{"bash"}},
	})
	assert.False(t, f.PreFilter("bash"), "user include should override denylist")
	assert.True(t, f.PreFilter("sleep"), "non-included denylist entries still blocked")
}

func TestPreFilterUserExclude(t *testing.T) {
	f := newProcessFilter(FilterConfig{
		ExcludeSystemProcesses: false,
		MinLifetimeScans:       2,
		Exclude:                MatchConfig{ExecutableNames: []string{"my-noise-tool"}},
	})
	assert.True(t, f.PreFilter("my-noise-tool"))
	assert.False(t, f.PreFilter("my-app"))
}

func TestShouldIncludeIdentifiedRuntimeAlwaysPasses(t *testing.T) {
	f := newProcessFilter(FilterConfig{ExcludeSystemProcesses: true, MinLifetimeScans: 2})
	s := ProcessSnapshot{
		ProcessDescription: ProcessDescription{ExecutableName: "bash"},
		RuntimeEvidence:    RuntimeEvidence{RuntimeStatus: "identified", RuntimeFamily: "ruby"},
	}
	assert.True(t, f.ShouldInclude(s), "identified runtime should always be included")
}

func TestShouldIncludeInstrumentedAlwaysPasses(t *testing.T) {
	f := newProcessFilter(FilterConfig{ExcludeSystemProcesses: true, MinLifetimeScans: 2})
	s := ProcessSnapshot{
		ProcessDescription:      ProcessDescription{ExecutableName: "my-app"},
		RuntimeEvidence:         RuntimeEvidence{RuntimeStatus: "unknown"},
		InstrumentationEvidence: InstrumentationEvidence{Status: "detected"},
	}
	assert.True(t, f.ShouldInclude(s), "instrumented process should always be included")
}

func TestShouldIncludeExcludesSystemProcess(t *testing.T) {
	f := newProcessFilter(FilterConfig{ExcludeSystemProcesses: true, MinLifetimeScans: 2})
	s := ProcessSnapshot{
		ProcessDescription: ProcessDescription{ExecutableName: "sleep"},
		RuntimeEvidence:    RuntimeEvidence{RuntimeStatus: "unknown"},
	}
	assert.False(t, f.ShouldInclude(s))
}

func TestShouldIncludeUserExcludeByPath(t *testing.T) {
	f := newProcessFilter(FilterConfig{
		ExcludeSystemProcesses: false,
		MinLifetimeScans:       2,
		Exclude:                MatchConfig{ExecutablePaths: []string{"/usr/libexec/*"}},
	})
	s := ProcessSnapshot{
		ProcessDescription: ProcessDescription{
			ExecutableName: "helper",
			ExecutablePath: "/usr/libexec/helper",
		},
		RuntimeEvidence: RuntimeEvidence{RuntimeStatus: "unknown"},
	}
	assert.False(t, f.ShouldInclude(s))
}

func TestShouldIncludeUserIncludeOverridesExclude(t *testing.T) {
	f := newProcessFilter(FilterConfig{
		ExcludeSystemProcesses: true,
		MinLifetimeScans:       2,
		Include:                MatchConfig{ExecutableNames: []string{"sleep"}},
	})
	s := ProcessSnapshot{
		ProcessDescription: ProcessDescription{ExecutableName: "sleep"},
		RuntimeEvidence:    RuntimeEvidence{RuntimeStatus: "unknown"},
	}
	assert.True(t, f.ShouldInclude(s), "user include should override system denylist")
}

func TestShouldIncludeUserIncludeByPath(t *testing.T) {
	f := newProcessFilter(FilterConfig{
		ExcludeSystemProcesses: true,
		MinLifetimeScans:       2,
		Include:                MatchConfig{ExecutablePaths: []string{"/opt/myapp/*"}},
	})
	s := ProcessSnapshot{
		ProcessDescription: ProcessDescription{
			ExecutableName: "sleep",
			ExecutablePath: "/opt/myapp/sleep",
		},
		RuntimeEvidence: RuntimeEvidence{RuntimeStatus: "unknown"},
	}
	assert.True(t, f.ShouldInclude(s))
}

func TestShouldIncludeUnknownProcessNotInDenylist(t *testing.T) {
	f := newProcessFilter(FilterConfig{ExcludeSystemProcesses: true, MinLifetimeScans: 2})
	s := ProcessSnapshot{
		ProcessDescription: ProcessDescription{ExecutableName: "custom-daemon"},
		RuntimeEvidence:    RuntimeEvidence{RuntimeStatus: "unknown"},
	}
	assert.True(t, f.ShouldInclude(s), "unknown process not in denylist should be included")
}

func TestSeenTwiceFilterForUnidentifiedProcesses(t *testing.T) {
	state := newProcessState()
	unknown := ProcessSnapshot{
		ProcessIdentity:       ProcessIdentity{Key: ProcessKey{PID: 42, StartTime: 100}},
		ProcessDescription:    ProcessDescription{Command: "custom-daemon", ExecutableName: "custom-daemon"},
		RuntimeEvidence:       RuntimeEvidence{RuntimeStatus: "unknown"},
		ObservationProvenance: ObservationProvenance{Source: "procfs"},
	}

	events, _ := state.apply(ScanResult{Processes: []ProcessSnapshot{unknown}, Complete: true}, 1, 10, 2, 0)
	assert.Empty(t, events, "unidentified process should not be admitted on first scan")
	assert.Len(t, state.pending, 1, "should be pending")

	events, _ = state.apply(ScanResult{Processes: []ProcessSnapshot{unknown}, Complete: true}, 1, 10, 2, 0)
	assert.Len(t, events, 1, "unidentified process should be admitted on second scan")
	assert.Equal(t, "process.started", events[0].Name)
	assert.Empty(t, state.pending, "pending should be cleared after admission")
}

func TestSeenTwiceIdentifiedBypassesPending(t *testing.T) {
	state := newProcessState()
	java := snapshot(42, 100, "java")

	events, _ := state.apply(ScanResult{Processes: []ProcessSnapshot{java}, Complete: true}, 1, 10, 2, 0)
	require.Len(t, events, 1, "identified process should be admitted immediately")
	assert.Equal(t, "process.started", events[0].Name)
}

func TestSeenTwiceInstrumentedBypassesPending(t *testing.T) {
	state := newProcessState()
	instrumented := ProcessSnapshot{
		ProcessIdentity:         ProcessIdentity{Key: ProcessKey{PID: 42, StartTime: 100}},
		ProcessDescription:      ProcessDescription{Command: "my-app", ExecutableName: "my-app"},
		RuntimeEvidence:         RuntimeEvidence{RuntimeStatus: "unknown"},
		InstrumentationEvidence: InstrumentationEvidence{Status: "detected"},
		ObservationProvenance:   ObservationProvenance{Source: "procfs"},
	}

	events, _ := state.apply(ScanResult{Processes: []ProcessSnapshot{instrumented}, Complete: true}, 1, 10, 2, 0)
	require.Len(t, events, 1, "instrumented process should be admitted immediately")
	assert.Equal(t, "process.started", events[0].Name)
}

func TestPendingEvictedWhenProcessDisappears(t *testing.T) {
	state := newProcessState()
	unknown := ProcessSnapshot{
		ProcessIdentity:       ProcessIdentity{Key: ProcessKey{PID: 42, StartTime: 100}},
		ProcessDescription:    ProcessDescription{Command: "ephemeral"},
		RuntimeEvidence:       RuntimeEvidence{RuntimeStatus: "unknown"},
		ObservationProvenance: ObservationProvenance{Source: "procfs"},
	}

	state.apply(ScanResult{Processes: []ProcessSnapshot{unknown}, Complete: true}, 1, 10, 2, 0)
	assert.Len(t, state.pending, 1)

	state.apply(ScanResult{Complete: true}, 1, 10, 2, 0)
	assert.Empty(t, state.pending, "pending should be evicted when process disappears")
}
