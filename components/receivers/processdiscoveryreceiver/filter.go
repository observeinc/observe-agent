package processdiscoveryreceiver

import (
	"path/filepath"
	"strings"
)

// Processes in systemDenylist are unconditionally dropped when
// ExcludeSystemProcesses is enabled. The check matches the short
// /proc/<pid>/comm name (max 15 chars), so entries must respect
// that limit. Prefix entries ending with "*" are matched via
// strings.HasPrefix.
var systemDenylist = []string{
	// Container infrastructure
	"pause",
	"runc",
	"dumb-init",
	"tini",
	"containerd-shim*",

	// Shells
	"bash",
	"sh",
	"zsh",
	"dash",
	"fish",
	"csh",
	"tcsh",
	"ksh",

	// System daemons
	"systemd",
	"systemd-*",
	"crond",
	"cron",
	"dbus-broker",
	"dbus-broker-lau*",
	"journalctl",

	// Coreutils / ephemeral utilities
	"sleep",
	"flock",
	"timeout",
	"env",
	"nice",
	"nohup",
	"stdbuf",
	"xargs",
	"sudo",
	"su",

	// Build tools
	"gcc",
	"cc1",
	"cc1plus",
	"collect2",
	"ld",
	"ld.bfd",
	"ld.gold",
	"make",
	"cmake",
	"as",
}

type processFilter struct {
	excludeSystem bool
	denySet       map[string]struct{}
	denyPrefixes  []string
	includeNames  map[string]struct{}
	excludeNames  map[string]struct{}
	includePaths  []string
	excludePaths  []string
}

func newProcessFilter(cfg FilterConfig) *processFilter {
	f := &processFilter{
		excludeSystem: cfg.ExcludeSystemProcesses,
		denySet:       make(map[string]struct{}),
		includeNames:  make(map[string]struct{}),
		excludeNames:  make(map[string]struct{}),
		includePaths:  cfg.Include.ExecutablePaths,
		excludePaths:  cfg.Exclude.ExecutablePaths,
	}
	if cfg.ExcludeSystemProcesses {
		for _, entry := range systemDenylist {
			if strings.HasSuffix(entry, "*") {
				f.denyPrefixes = append(f.denyPrefixes, strings.TrimSuffix(entry, "*"))
			} else {
				f.denySet[entry] = struct{}{}
			}
		}
	}
	for _, name := range cfg.Include.ExecutableNames {
		f.includeNames[name] = struct{}{}
	}
	for _, name := range cfg.Exclude.ExecutableNames {
		f.excludeNames[name] = struct{}{}
	}
	return f
}

// PreFilter performs a cheap check on the /proc/<pid>/comm name before
// the expensive full Inspect. Returns true if the process should be
// skipped (filtered out).
func (f *processFilter) PreFilter(comm string) bool {
	if _, ok := f.includeNames[comm]; ok {
		return false
	}
	if f.isSystemDenied(comm) {
		return true
	}
	if _, ok := f.excludeNames[comm]; ok {
		return true
	}
	return false
}

// ShouldInclude decides whether a fully-inspected process should be
// tracked. Processes with an identified runtime or detected
// instrumentation always pass. User include patterns override excludes.
func (f *processFilter) ShouldInclude(snapshot ProcessSnapshot) bool {
	if f.matchesInclude(snapshot) {
		return true
	}
	if snapshot.RuntimeStatus == "identified" {
		return true
	}
	if snapshot.InstrumentationEvidence.Status == "detected" {
		return true
	}
	if f.matchesExclude(snapshot) {
		return false
	}
	if f.isSystemDenied(snapshot.ExecutableName) {
		return false
	}
	return true
}

func (f *processFilter) isSystemDenied(name string) bool {
	if !f.excludeSystem {
		return false
	}
	if _, ok := f.denySet[name]; ok {
		return true
	}
	for _, prefix := range f.denyPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func (f *processFilter) matchesInclude(snapshot ProcessSnapshot) bool {
	if _, ok := f.includeNames[snapshot.ExecutableName]; ok {
		return true
	}
	for _, pattern := range f.includePaths {
		if matched, _ := filepath.Match(pattern, snapshot.ExecutablePath); matched {
			return true
		}
	}
	return false
}

func (f *processFilter) matchesExclude(snapshot ProcessSnapshot) bool {
	if _, ok := f.excludeNames[snapshot.ExecutableName]; ok {
		return true
	}
	for _, pattern := range f.excludePaths {
		if matched, _ := filepath.Match(pattern, snapshot.ExecutablePath); matched {
			return true
		}
	}
	return false
}
