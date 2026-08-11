//go:build linux

package processdiscoveryreceiver

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiscoverNetworkEndpointsGroupsByPortAndState(t *testing.T) {
	root := t.TempDir()
	pidDir := filepath.Join(root, "10")
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "fd"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(pidDir, "net"), 0o755))
	require.NoError(t, os.Symlink("socket:[42]", filepath.Join(pidDir, "fd", "3")))
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp"), []byte(
		"  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"+
			"   0: 00000000:1F90 00000000:0000 0A 0:0 00:0 0 0 0 0 42\n"+
			"   1: 0100007F:1F90 0100007F:C350 01 0:0 00:0 0 0 0 0 42\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(pidDir, "net", "tcp6"), []byte("  sl  local_address rem_address   st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n"), 0o600))

	endpoints, status := discoverNetworkEndpoints(root, 10, 10)
	assert.Equal(t, "complete", status)
	require.Len(t, endpoints, 2)
	assert.Equal(t, 8080, endpoints[0].LocalPort)
	assert.Equal(t, "established", endpoints[0].State)
	assert.Equal(t, "listen", endpoints[1].State)
}

func TestParseProcNetPort(t *testing.T) {
	port, err := parseProcNetPort("00000000:1F90")
	require.NoError(t, err)
	assert.Equal(t, 8080, port)
}
