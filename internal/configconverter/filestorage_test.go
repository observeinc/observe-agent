package configconverter

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestMigrateFileStorage_RenamesLegacyOwnerFiles(t *testing.T) {
	dir := t.TempDir()
	legacyQueue := filepath.Join(dir, "exporter_otlphttp_observe_logs")
	legacyCheckpoints := filepath.Join(dir, "receiver_filelog_host_monitoring")
	require.NoError(t, os.WriteFile(legacyQueue, []byte("queued"), 0o600))
	require.NoError(t, os.WriteFile(legacyCheckpoints, []byte("offsets"), 0o600))
	// Unrelated files must be left alone.
	other := filepath.Join(dir, "exporter_otlphttp_mine_logs")
	require.NoError(t, os.WriteFile(other, []byte("custom"), 0o600))

	require.NoError(t, MigrateFileStorage(dir, zap.NewNop()))

	assert.NoFileExists(t, legacyQueue)
	assert.FileExists(t, filepath.Join(dir, "exporter_otlp_http_observe_logs"))
	got, err := os.ReadFile(filepath.Join(dir, "exporter_otlp_http_observe_logs"))
	require.NoError(t, err)
	assert.Equal(t, "queued", string(got))

	assert.NoFileExists(t, legacyCheckpoints)
	assert.FileExists(t, filepath.Join(dir, "receiver_file_log_host_monitoring"))

	assert.FileExists(t, other)
}

func TestMigrateFileStorage_LeavesExistingCanonicalFileAlone(t *testing.T) {
	dir := t.TempDir()
	legacyQueue := filepath.Join(dir, "exporter_otlphttp_observe_logs")
	canonicalQueue := filepath.Join(dir, "exporter_otlp_http_observe_logs")
	require.NoError(t, os.WriteFile(legacyQueue, []byte("old"), 0o600))
	require.NoError(t, os.WriteFile(canonicalQueue, []byte("new"), 0o600))

	core, logs := observer.New(zap.WarnLevel)
	require.NoError(t, MigrateFileStorage(dir, zap.New(core)))

	got, err := os.ReadFile(canonicalQueue)
	require.NoError(t, err)
	assert.Equal(t, "new", string(got), "must not overwrite a queue already opened under the new id")
	assert.FileExists(t, legacyQueue)
	assert.NotZero(t, logs.Len())
}

func TestMigrateFileStorage_MissingDirIsNoOp(t *testing.T) {
	require.NoError(t, MigrateFileStorage(filepath.Join(t.TempDir(), "does-not-exist"), zap.NewNop()))
	require.NoError(t, MigrateFileStorage("", zap.NewNop()))
}

func TestFileStorageFilename_MatchesFilestorageOwnerPattern(t *testing.T) {
	assert.Equal(t, "exporter_otlphttp_observe_logs", fileStorageFilename("exporter", "otlphttp", "observe", "logs"))
	assert.Equal(t, "exporter_otlp_http_observe_logs", fileStorageFilename("exporter", "otlp_http", "observe", "logs"))
	assert.Equal(t, "receiver_filelog_host_monitoring", fileStorageFilename("receiver", "filelog", "host_monitoring", ""))
	assert.Equal(t, "receiver_file_log_host_monitoring", fileStorageFilename("receiver", "file_log", "host_monitoring", ""))
}
