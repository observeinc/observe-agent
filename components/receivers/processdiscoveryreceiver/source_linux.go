//go:build linux

package processdiscoveryreceiver

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type procFSSource struct {
	root               string
	maxCmdlineBytes    int64
	detector           runtimeDetector
	cache              *executableCache
	bootTime           time.Time
	clockTicks         uint64
	kubernetes         processEnricher
	scrubber           argumentScrubber
	commandLineEnabled bool
	maxPortsPerProcess int
	networkEnabled     bool
}

type procStat struct {
	pid              int32
	parentPID        int32
	groupLeaderPID   int32
	sessionLeaderPID int32
	startTime        uint64
	state            string
}

type procStatus struct {
	effectiveUserID int64
	virtualPID      int32
}

func platformSupported() bool { return true }

func newProcessSource(cfg *Config) (ProcessSource, error) {
	bootTime, err := readBootTime(filepath.Join(cfg.ProcFSPath, "stat"))
	if err != nil {
		return nil, fmt.Errorf("read Linux boot time: %w", err)
	}
	return &procFSSource{
		root: cfg.ProcFSPath, maxCmdlineBytes: int64(cfg.MaxCmdlineBytes),
		detector: newRuntimeDetector(cfg.Runtimes),
		cache:    newExecutableCache(cfg.RuntimeDetection.BinaryCacheTTL, cfg.RuntimeDetection.BinaryCacheSize),
		bootTime: bootTime, clockTicks: cfg.ClockTicks,
		kubernetes: newKubernetesEnricher(cfg.Kubernetes),
		scrubber:   newArgumentScrubber(cfg.CommandLine), commandLineEnabled: cfg.CommandLine.Enabled,
		maxPortsPerProcess: cfg.Network.MaxPortsPerProcess,
		networkEnabled:     cfg.Network.Enabled,
	}, nil
}

func (s *procFSSource) Scan(ctx context.Context) (ScanResult, error) {
	if s.kubernetes != nil {
		s.kubernetes.Refresh(ctx)
	}
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return ScanResult{}, fmt.Errorf("read procfs root: %w", err)
	}
	var pids []int
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if entry.IsDir() && err == nil && pid > 0 {
			pids = append(pids, pid)
		}
	}
	sort.Ints(pids)
	result := ScanResult{Complete: true, UnavailablePIDs: make(map[int32]struct{})}
	for _, pid := range pids {
		if err := ctx.Err(); err != nil {
			result.Complete = false
			return result, err
		}
		snapshot, err := s.Inspect(ctx, int32(pid))
		if err != nil {
			if !errors.Is(err, fs.ErrNotExist) {
				result.UnavailablePIDs[int32(pid)] = struct{}{}
				result.Complete = false
			}
			continue
		}
		result.Processes = append(result.Processes, snapshot)
	}
	return result, nil
}

func (s *procFSSource) Inspect(_ context.Context, pid int32) (ProcessSnapshot, error) {
	pidDir := filepath.Join(s.root, strconv.Itoa(int(pid)))
	statPath := filepath.Join(pidDir, "stat")
	beforeBytes, err := os.ReadFile(statPath)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	before, err := parseProcStat(beforeBytes)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	cmdline, complete, cmdlineErr := readCmdline(filepath.Join(pidDir, "cmdline"), s.maxCmdlineBytes)
	exeLink := filepath.Join(pidDir, "exe")
	exePath := readLink(exeLink)
	exeBase := filepath.Base(exePath)

	command := exeBase
	if len(cmdline) > 0 {
		command = filepath.Base(cmdline[0])
	}
	result, found := s.detector.detectCommand(command, cmdline)
	if !found && exeBase != command {
		result, found = s.detector.detectCommand(exeBase, cmdline)
	}
	if exeResult, exeFound := s.detector.detectCommand(exeBase, cmdline); exeFound {
		mergeDetectionResult(&result, &found, exeResult)
	}

	identity, identityErr := linuxExecutableIdentity(exeLink)
	statusData, _ := readProcStatus(filepath.Join(pidDir, "status"))
	owner := lookupUsername(statusData.effectiveUserID)
	workingDirectory := readLink(filepath.Join(pidDir, "cwd"))
	cgroupData, cgroupErr := os.ReadFile(filepath.Join(pidDir, "cgroup"))
	var executableErr error
	if executableResult, executableFound, _, err := s.inspectExecutable(exeLink); err != nil {
		executableErr = err
	} else if executableFound {
		mergeDetectionResult(&result, &found, executableResult)
	}
	libraryResult, libraryFound, mapsErr := detectMappedRuntime(filepath.Join(pidDir, "maps"), s.detector.enabled)
	if libraryFound {
		mergeDetectionResult(&result, &found, libraryResult)
	}

	afterBytes, err := os.ReadFile(statPath)
	if err != nil {
		return ProcessSnapshot{}, err
	}
	after, err := parseProcStat(afterBytes)
	if err != nil || before.pid != after.pid || before.startTime != after.startTime {
		return ProcessSnapshot{}, fmt.Errorf("process identity changed during inspection")
	}
	status := "unknown"
	if found {
		status = "identified"
	} else if errors.Is(cmdlineErr, fs.ErrPermission) || errors.Is(identityErr, fs.ErrPermission) || errors.Is(executableErr, fs.ErrPermission) || errors.Is(mapsErr, fs.ErrPermission) {
		status = "inaccessible"
	} else if cmdlineErr != nil || !complete {
		status = "unavailable"
	}
	snapshot := ProcessSnapshot{
		ProcessIdentity: ProcessIdentity{Key: ProcessKey{
			PID: before.pid, StartTime: before.startTime, CreationTime: s.creationTime(before.startTime),
		}},
		ProcessDescription: ProcessDescription{
			ParentPID: before.parentPID, GroupLeaderPID: before.groupLeaderPID,
			SessionLeaderPID: before.sessionLeaderPID, VirtualPID: statusData.virtualPID,
			Command: command, ExecutableName: exeBase, ExecutablePath: exePath,
			WorkingDirectory: workingDirectory, Owner: owner, UserID: statusData.effectiveUserID,
			State: normalizeProcessState(before.state), GNUBuildID: identity.GNUBuildID,
			GoBuildID: identity.GoBuildID, ApplicationEntrypoint: applicationEntrypoint(result.runtimeFamily, cmdline),
			ExecutableArch: identity.Architecture,
		},
		RuntimeEvidence: RuntimeEvidence{
			RuntimeName: result.runtimeName, RuntimeFamily: result.runtimeFamily, RuntimeVersion: result.runtimeVersion,
			InferredVersion: result.inferredVersion, InferredVersionKind: result.inferredVersionKind,
			RuntimeAssertions: result.assertions, RuntimeStatus: status,
		},
		ObservationProvenance: ObservationProvenance{Source: "procfs", ObservedAt: time.Now(), Complete: complete},
	}
	if s.commandLineEnabled {
		snapshot.CommandArgs = s.scrubber.scrub(cmdline)
	}
	if cgroupErr == nil {
		snapshot.Cgroup = strings.TrimSpace(string(cgroupData))
		snapshot.ContainerID, snapshot.K8sPodUID = parseCgroupIdentityForPID(cgroupData, pid)
	}
	if s.kubernetes != nil {
		s.kubernetes.Enrich(&snapshot)
	}
	if s.networkEnabled {
		endpoints, status := discoverNetworkEndpoints(s.root, pid, s.maxPortsPerProcess)
		snapshot.NetworkEvidence = NetworkEvidence{Endpoints: endpoints, Status: status}
	}
	return snapshot, nil
}

func (s *procFSSource) inspectExecutable(path string) (detectionResult, bool, executableIdentity, error) {
	identity, err := linuxExecutableIdentity(path)
	if err != nil {
		return detectionResult{}, false, executableIdentity{}, err
	}
	if cached, found, ok := s.cache.get(identity, time.Now()); ok {
		return cached, found, identity, nil
	}
	result, found, _, err := inspectLinuxExecutable(path, s.detector.enabled)
	if err == nil {
		s.cache.put(identity, result, found, time.Now())
	}
	return result, found, identity, err
}

func detectMappedRuntime(path string, enabled map[string]struct{}) (detectionResult, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return detectionResult{}, false, err
	}
	defer file.Close()
	scanner := bufio.NewScanner(io.LimitReader(file, 4*1024*1024))
	for scanner.Scan() {
		line := strings.ToLower(scanner.Text())
		checks := []struct{ runtime, needle string }{
			{"java", "libjvm.so"}, {"dotnet", "libcoreclr.so"}, {"python", "libpython"},
			{"ruby", "libruby"}, {"erlang", "beam.smp"}, {"php", "libphp"},
		}
		for _, check := range checks {
			if _, ok := enabled[check.runtime]; ok && strings.Contains(line, check.needle) {
				return detectionResult{runtimeFamily: check.runtime, assertions: []string{check.runtime + ".loaded_library"}}, true, nil
			}
		}
		if strings.Contains(line, "libstdc++.so") {
			return detectionResult{runtimeFamily: "native", assertions: []string{"native.libstdcxx_loaded"}}, true, nil
		}
		if strings.Contains(line, "libc++.so") || strings.Contains(line, "libc++abi.so") {
			return detectionResult{runtimeFamily: "native", assertions: []string{"native.libcxx_loaded"}}, true, nil
		}
	}
	return detectionResult{}, false, scanner.Err()
}

func (s *procFSSource) creationTime(startTicks uint64) time.Time {
	if s.bootTime.IsZero() || s.clockTicks == 0 {
		return time.Time{}
	}
	seconds := time.Duration(startTicks/s.clockTicks) * time.Second
	nanos := time.Duration(startTicks%s.clockTicks) * time.Second / time.Duration(s.clockTicks)
	return s.bootTime.Add(seconds + nanos)
}

func readBootTime(path string) (time.Time, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return time.Time{}, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(line, "btime ") {
			continue
		}
		seconds, err := strconv.ParseInt(strings.TrimSpace(strings.TrimPrefix(line, "btime ")), 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		return time.Unix(seconds, 0), nil
	}
	return time.Time{}, fmt.Errorf("btime not found")
}

func parseProcStat(data []byte) (procStat, error) {
	line := strings.TrimSpace(string(data))
	open, close := strings.IndexByte(line, '('), strings.LastIndex(line, ") ")
	if open <= 0 || close <= open {
		return procStat{}, fmt.Errorf("malformed proc stat")
	}
	pidValue, err := strconv.ParseInt(strings.TrimSpace(line[:open]), 10, 32)
	if err != nil {
		return procStat{}, fmt.Errorf("parse pid: %w", err)
	}
	fields := strings.Fields(line[close+2:])
	if len(fields) < 20 {
		return procStat{}, fmt.Errorf("proc stat has %d fields after comm, need 20", len(fields))
	}
	parentPID, err := strconv.ParseInt(fields[1], 10, 32)
	if err != nil {
		return procStat{}, fmt.Errorf("parse parent pid: %w", err)
	}
	startTime, err := strconv.ParseUint(fields[19], 10, 64)
	if err != nil {
		return procStat{}, fmt.Errorf("parse start time: %w", err)
	}
	groupLeaderPID, _ := strconv.ParseInt(fields[2], 10, 32)
	sessionLeaderPID, _ := strconv.ParseInt(fields[3], 10, 32)
	return procStat{
		pid: int32(pidValue), parentPID: int32(parentPID), groupLeaderPID: int32(groupLeaderPID),
		sessionLeaderPID: int32(sessionLeaderPID), startTime: startTime, state: fields[0],
	}, nil
}

func readCmdline(path string, maxBytes int64) ([]string, bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, false, err
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, maxBytes+1))
	if err != nil {
		return nil, false, err
	}
	complete := int64(len(data)) <= maxBytes
	if !complete {
		data = data[:maxBytes]
	}
	parts := bytes.Split(data, []byte{0})
	if !complete && len(parts) > 0 {
		parts = parts[:len(parts)-1]
	}
	args := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) > 0 {
			args = append(args, string(part))
		}
	}
	return args, complete, nil
}

func readLink(path string) string {
	target, err := os.Readlink(path)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(target, " (deleted)")
}

func readExecutableBase(path string) string { return filepath.Base(readLink(path)) }

func readProcStatus(path string) (procStatus, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return procStatus{}, err
	}
	result := procStatus{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		switch strings.TrimSuffix(fields[0], ":") {
		case "Uid":
			if len(fields) > 2 {
				result.effectiveUserID, _ = strconv.ParseInt(fields[2], 10, 64)
			}
		case "NSpid":
			value, _ := strconv.ParseInt(fields[len(fields)-1], 10, 32)
			result.virtualPID = int32(value)
		}
	}
	return result, nil
}

func lookupUsername(uid int64) string {
	account, err := user.LookupId(strconv.FormatInt(uid, 10))
	if err != nil {
		return ""
	}
	return account.Username
}

func normalizeProcessState(state string) string {
	switch state {
	case "R":
		return "running"
	case "S", "D", "I":
		return "sleeping"
	case "T", "t":
		return "stopped"
	case "Z", "X", "x":
		return "defunct"
	default:
		return state
	}
}

func applicationEntrypoint(runtimeFamily string, args []string) string {
	if len(args) < 2 {
		return ""
	}
	switch runtimeFamily {
	case "java":
		for index := 1; index < len(args); index++ {
			if args[index] == "-jar" && index+1 < len(args) {
				return args[index+1]
			}
			if !strings.HasPrefix(args[index], "-") {
				return args[index]
			}
		}
	case "python", "ruby", "nodejs", "php", "perl":
		for _, argument := range args[1:] {
			if !strings.HasPrefix(argument, "-") {
				return argument
			}
		}
	case "dotnet":
		return args[1]
	}
	return ""
}
