//go:build linux

package processdiscoveryreceiver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseProcStatWithParentheses(t *testing.T) {
	parsed, err := parseProcStat([]byte(procStatLine(42, "worker (pool)", 7, 12345)))
	require.NoError(t, err)
	assert.Equal(t, int32(42), parsed.pid)
	assert.Equal(t, int32(7), parsed.parentPID)
	assert.Equal(t, uint64(12345), parsed.startTime)
}

func TestProcFSSourceIncludesUnknownProcesses(t *testing.T) {
	root := t.TempDir()
	writeProcessFixture(t, root, 10, 1, 100, []string{"/usr/bin/java", "-jar", "app.jar"})
	writeProcessFixture(t, root, 11, 1, 101, []string{"/usr/sbin/native-app"})
	source := testProcSource(root)
	result, err := source.Scan(t.Context())
	require.NoError(t, err)
	require.Len(t, result.Processes, 2)
	assert.Equal(t, "java", result.Processes[0].RuntimeFamily)
	assert.Equal(t, "unknown", result.Processes[1].RuntimeStatus)
}

func TestProcFSSourceUsesMatchingExecutableVersionEvidence(t *testing.T) {
	root := t.TempDir()
	writeProcessFixture(t, root, 10, 1, 100, []string{"python", "app.py"})
	require.NoError(t, os.Symlink("/usr/bin/python3.12", filepath.Join(root, "10", "exe")))
	snapshot, err := testProcSource(root).Inspect(t.Context(), 10)
	require.NoError(t, err)
	assert.Equal(t, "python", snapshot.RuntimeFamily)
	assert.Equal(t, "3.12", snapshot.InferredVersion)
}

func TestProcFSSourceRejectsConflictingExecutableEvidence(t *testing.T) {
	root := t.TempDir()
	writeProcessFixture(t, root, 10, 1, 100, []string{"java", "Main"})
	require.NoError(t, os.Symlink("/usr/bin/python3.12", filepath.Join(root, "10", "exe")))
	snapshot, err := testProcSource(root).Inspect(t.Context(), 10)
	require.NoError(t, err)
	assert.Equal(t, "java", snapshot.RuntimeFamily)
	assert.Empty(t, snapshot.InferredVersion)
	assert.NotContains(t, snapshot.RuntimeAssertions, "python.executable")
}

func TestReadCmdlineBounded(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cmdline")
	require.NoError(t, os.WriteFile(path, []byte("python3\x00"+strings.Repeat("x", 100)), 0o600))
	args, complete, err := readCmdline(path, 8)
	require.NoError(t, err)
	assert.False(t, complete)
	require.Len(t, args, 1)
	assert.Equal(t, "python3", args[0])
}

func TestReadBootTimeAndCreationTime(t *testing.T) {
	path := filepath.Join(t.TempDir(), "stat")
	require.NoError(t, os.WriteFile(path, []byte("cpu 1 2 3 4\nbtime 1000\n"), 0o600))
	bootTime, err := readBootTime(path)
	require.NoError(t, err)
	assert.Equal(t, time.Unix(1000, 0), bootTime)
	source := &procFSSource{bootTime: bootTime, clockTicks: 100}
	assert.Equal(t, time.Unix(1002, 500_000_000), source.creationTime(250))
}

func TestMappedNativeLibraryIsAssertionNotCPlusPlus(t *testing.T) {
	path := filepath.Join(t.TempDir(), "maps")
	require.NoError(t, os.WriteFile(path, []byte("7f00-7f01 r-xp 0 00:00 0 /usr/lib/libstdc++.so.6\n"), 0o600))
	result, found, err := detectMappedRuntime(path, map[string]struct{}{})
	require.NoError(t, err)
	require.True(t, found)
	assert.Equal(t, "native", result.runtimeFamily)
	assert.Equal(t, []string{"native.libstdcxx_loaded"}, result.assertions)
}

func testProcSource(root string) *procFSSource {
	return &procFSSource{
		root: root, maxCmdlineBytes: 8192, detector: newRuntimeDetector(supportedRuntimes),
		cache: newExecutableCache(time.Minute, 10), bootTime: time.Unix(1000, 0), clockTicks: 100,
	}
}

func writeProcessFixture(t *testing.T, root string, pid, ppid int, start uint64, args []string) {
	t.Helper()
	dir := filepath.Join(root, fmt.Sprint(pid))
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "stat"), []byte(procStatLine(pid, "fixture", ppid, start)), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cmdline"), []byte(strings.Join(args, "\x00")+"\x00"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cgroup"), nil, 0o600))
}

func procStatLine(pid int, comm string, ppid int, start uint64) string {
	fields := []string{"S", fmt.Sprint(ppid)}
	for len(fields) < 19 {
		fields = append(fields, "0")
	}
	fields = append(fields, fmt.Sprint(start))
	return fmt.Sprintf("%d (%s) %s\n", pid, comm, strings.Join(fields, " "))
}
