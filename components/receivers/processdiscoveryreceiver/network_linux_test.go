//go:build linux

package processdiscoveryreceiver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectOTLPConnectionsMatchesStandardPort(t *testing.T) {
	root := t.TempDir()
	pidDir := filepath.Join(root, "10")
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "fd"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "net"), 0o755))
	require.NoError(t, os.Symlink("socket:[42]", filepath.Join(pidDir, "fd", "3")))
	// Remote 127.0.0.1:4317 (hex 0100007F:10DD), ESTABLISHED (01), inode 42
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"+
			"   0: 00000000:C350 0100007F:10DD 01 00000000:00000000 00:00000000 00000000 0 0 42\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp6"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"), 0o600))

	evidence := detectOTLPConnections(root, 10, nil, 128)
	assert.Equal(t, "detected", evidence.Status)
	require.Len(t, evidence.OTLPConnections, 1)
	assert.Equal(t, "127.0.0.1", evidence.OTLPConnections[0].RemoteHost)
	assert.Equal(t, 4317, evidence.OTLPConnections[0].RemotePort)
	assert.Equal(t, "standard_port:4317", evidence.OTLPConnections[0].MatchedRule)
}

func TestDetectOTLPConnectionsIgnoresNonOTLPPorts(t *testing.T) {
	root := t.TempDir()
	pidDir := filepath.Join(root, "10")
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "fd"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "net"), 0o755))
	require.NoError(t, os.Symlink("socket:[42]", filepath.Join(pidDir, "fd", "3")))
	// Remote 127.0.0.1:8080 (hex 0100007F:1F90), ESTABLISHED, inode 42
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"+
			"   0: 00000000:C350 0100007F:1F90 01 00000000:00000000 00:00000000 00000000 0 0 42\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp6"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"), 0o600))

	evidence := detectOTLPConnections(root, 10, nil, 128)
	assert.Equal(t, "none_detected", evidence.Status)
	assert.Empty(t, evidence.OTLPConnections)
}

func TestDetectOTLPConnectionsMatchesConfiguredEndpoint(t *testing.T) {
	root := t.TempDir()
	pidDir := filepath.Join(root, "10")
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "fd"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "net"), 0o755))
	require.NoError(t, os.Symlink("socket:[42]", filepath.Join(pidDir, "fd", "3")))
	// Remote 127.0.0.1:9999 (hex 0100007F:270F), ESTABLISHED, inode 42
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"+
			"   0: 00000000:C350 0100007F:270F 01 00000000:00000000 00:00000000 00000000 0 0 42\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp6"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"), 0o600))

	endpoints := []otlpEndpoint{{Host: "127.0.0.1", Port: 9999}}
	evidence := detectOTLPConnections(root, 10, endpoints, 128)
	assert.Equal(t, "detected", evidence.Status)
	require.Len(t, evidence.OTLPConnections, 1)
	assert.Equal(t, "configured:127.0.0.1:9999", evidence.OTLPConnections[0].MatchedRule)
}

func TestDetectOTLPConnectionsIgnoresListenState(t *testing.T) {
	root := t.TempDir()
	pidDir := filepath.Join(root, "10")
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "fd"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "net"), 0o755))
	require.NoError(t, os.Symlink("socket:[42]", filepath.Join(pidDir, "fd", "3")))
	// Local 0.0.0.0:4317 in LISTEN (0A), inode 42 — should be ignored
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"+
			"   0: 00000000:10DD 00000000:0000 0A 00000000:00000000 00:00000000 00000000 0 0 42\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp6"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"), 0o600))

	evidence := detectOTLPConnections(root, 10, nil, 128)
	assert.Equal(t, "none_detected", evidence.Status)
	assert.Empty(t, evidence.OTLPConnections)
}

func TestParseProcNetAddressIPv4(t *testing.T) {
	ip, port, err := parseProcNetAddress("0100007F:10DD", 4)
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1", ip)
	assert.Equal(t, 4317, port)
}

func TestMatchOTLPEndpoint(t *testing.T) {
	assert.Equal(t, "standard_port:4317", matchOTLPEndpoint("10.0.0.1", 4317, nil))
	assert.Equal(t, "standard_port:4318", matchOTLPEndpoint("10.0.0.1", 4318, nil))
	assert.Equal(t, "", matchOTLPEndpoint("10.0.0.1", 8080, nil))

	endpoints := []otlpEndpoint{{Host: "10.0.0.1", Port: 8080}}
	assert.Equal(t, "configured:10.0.0.1:8080", matchOTLPEndpoint("10.0.0.1", 8080, endpoints))
}
