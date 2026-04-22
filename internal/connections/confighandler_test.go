package connections

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
	"gopkg.in/yaml.v3"
)

func collectTempDirsWithPrefix(t *testing.T, prefix string) []string {
	t.Helper()
	entries, err := os.ReadDir(os.TempDir())
	require.NoError(t, err)
	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), prefix) {
			dirs = append(dirs, filepath.Join(os.TempDir(), entry.Name()))
		}
	}
	return dirs
}

func setupTestViper(t *testing.T) {
	t.Helper()
	viper.Reset()
	viper.Set("token", "test:token123456789")
	viper.Set("observe_url", "https://example.observeinc.com")
	t.Cleanup(func() { viper.Reset() })
}

// observedLoggerCtx returns a context with a logger that records entries at
// debug+ so tests can assert on structured log output.
func observedLoggerCtx(t *testing.T) (context.Context, *observer.ObservedLogs) {
	t.Helper()
	core, logs := observer.New(zapcore.DebugLevel)
	l := zap.New(core)
	return logger.WithCtx(context.Background(), l), logs
}

// findLog returns the first log entry with the given message, or nil.
func findLog(logs *observer.ObservedLogs, msg string) *observer.LoggedEntry {
	for _, e := range logs.All() {
		if e.Message == msg {
			entry := e
			return &entry
		}
	}
	return nil
}

// TestSetupAndGetConfigs_DoesNotCreateTempDirs asserts that repeated calls
// to SetupAndGetConfigs do not create any observe-agent* directories in
// os.TempDir().
func TestSetupAndGetConfigs_DoesNotCreateTempDirs(t *testing.T) {
	setupTestViper(t)
	ctx := logger.WithCtx(context.Background(), logger.GetNop())

	before := collectTempDirsWithPrefix(t, TempFilesFolder)

	for i := 0; i < 3; i++ {
		fragments, err := SetupAndGetConfigs(ctx)
		require.NoError(t, err, "SetupAndGetConfigs call %d should succeed", i)
		require.NotEmpty(t, fragments)
		for _, f := range fragments {
			assert.NotEmpty(t, f.Name)
			assert.NotEmpty(t, f.Content)
		}
	}

	after := collectTempDirsWithPrefix(t, TempFilesFolder)
	assert.ElementsMatch(t, before, after,
		"SetupAndGetConfigs must not create any observe-agent* temp dirs (before=%v after=%v)", before, after)
}

// TestCleanupLegacyTempDirs_RemovesOrphans asserts CleanupLegacyTempDirs
// removes observe-agent*<numeric> directories whose contents are YAML
// fragments.
func TestCleanupLegacyTempDirs_RemovesOrphans(t *testing.T) {
	ctx := logger.WithCtx(context.Background(), logger.GetNop())

	orphans := make([]string, 0, 3)
	for i := 0; i < 3; i++ {
		dir, err := os.MkdirTemp("", TempFilesFolder)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(filepath.Join(dir, "fragment.yaml"), []byte("x: 1\n"), 0o600))
		orphans = append(orphans, dir)
	}
	t.Cleanup(func() {
		for _, d := range orphans {
			os.RemoveAll(d)
		}
	})

	CleanupLegacyTempDirs(ctx)

	for _, d := range orphans {
		_, err := os.Stat(d)
		assert.True(t, os.IsNotExist(err),
			"legacy temp dir %s should have been removed by CleanupLegacyTempDirs, got err=%v", d, err)
	}
}

// TestCleanupLegacyTempDirs_RunViaSetup asserts SetupAndGetConfigs invokes
// CleanupLegacyTempDirs.
func TestCleanupLegacyTempDirs_RunViaSetup(t *testing.T) {
	setupTestViper(t)
	ctx := logger.WithCtx(context.Background(), logger.GetNop())

	dir, err := os.MkdirTemp("", TempFilesFolder)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "fragment.yaml"), []byte("x: 1\n"), 0o600))
	t.Cleanup(func() { os.RemoveAll(dir) })

	_, err = SetupAndGetConfigs(ctx)
	require.NoError(t, err)

	_, err = os.Stat(dir)
	assert.True(t, os.IsNotExist(err),
		"legacy temp dir %s should have been cleaned up by SetupAndGetConfigs", dir)
}

// TestCleanupLegacyTempDirs_DoesNotRemoveNonLegacyPrefixedDirs asserts dirs
// sharing the observe-agent prefix but not matching the numeric-suffix
// naming format are preserved.
func TestCleanupLegacyTempDirs_DoesNotRemoveNonLegacyPrefixedDirs(t *testing.T) {
	ctx := logger.WithCtx(context.Background(), logger.GetNop())

	nonLegacy := filepath.Join(os.TempDir(), TempFilesFolder+"-unrelated-"+t.Name())
	require.NoError(t, os.Mkdir(nonLegacy, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(nonLegacy, "keep.txt"), []byte("keep"), 0o600))
	t.Cleanup(func() { os.RemoveAll(nonLegacy) })

	CleanupLegacyTempDirs(ctx)

	_, err := os.Stat(nonLegacy)
	assert.NoError(t, err, "non-legacy prefixed dir %s should not be removed", nonLegacy)
}

// TestCleanupLegacyTempDirs_DoesNotRemoveNumericPrefixedNonConfigDirs
// asserts numeric-suffixed dirs whose contents are not YAML-only are
// preserved.
func TestCleanupLegacyTempDirs_DoesNotRemoveNumericPrefixedNonConfigDirs(t *testing.T) {
	ctx := logger.WithCtx(context.Background(), logger.GetNop())

	nonConfig := filepath.Join(os.TempDir(), TempFilesFolder+"123456789")
	require.NoError(t, os.Mkdir(nonConfig, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(nonConfig, "keep.txt"), []byte("keep"), 0o600))
	t.Cleanup(func() { os.RemoveAll(nonConfig) })

	CleanupLegacyTempDirs(ctx)

	_, err := os.Stat(nonConfig)
	assert.NoError(t, err, "numeric prefixed non-config dir %s should not be removed", nonConfig)
}

// TestCleanupLegacyTempDirs_LogsStartAndSummary asserts start, per-removal,
// and summary log entries are emitted with their expected fields.
func TestCleanupLegacyTempDirs_LogsStartAndSummary(t *testing.T) {
	ctx, logs := observedLoggerCtx(t)
	root := t.TempDir()

	legacy := filepath.Join(root, TempFilesFolder+"42")
	require.NoError(t, os.Mkdir(legacy, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "fragment.yaml"), []byte("x: 1\n"), 0o600))

	cleanupLegacyTempDirsIn(ctx, root)

	start := findLog(logs, "scanning temp directory for legacy observe-agent config directories")
	require.NotNil(t, start, "expected a start log entry")
	assert.Equal(t, root, start.ContextMap()["temp_dir"])

	removed := findLog(logs, "removed legacy observe-agent temp directory")
	require.NotNil(t, removed, "expected a per-directory removal log entry")
	assert.Equal(t, legacy, removed.ContextMap()["path"])

	summary := findLog(logs, "legacy observe-agent temp directory cleanup finished")
	require.NotNil(t, summary, "expected a summary log entry")
	ctxMap := summary.ContextMap()
	assert.EqualValues(t, 1, ctxMap["removed"])
	assert.EqualValues(t, 0, ctxMap["failed"])
}

// TestCleanupLegacyTempDirs_LogsSkipReason asserts skipped candidates are
// logged with a non-empty `reason` field.
func TestCleanupLegacyTempDirs_LogsSkipReason(t *testing.T) {
	ctx, logs := observedLoggerCtx(t)
	root := t.TempDir()

	nonConfig := filepath.Join(root, TempFilesFolder+"7")
	require.NoError(t, os.Mkdir(nonConfig, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(nonConfig, "keep.txt"), []byte("keep"), 0o600))

	cleanupLegacyTempDirsIn(ctx, root)

	skip := findLog(logs, "skipping candidate temp directory")
	require.NotNil(t, skip, "expected a skip log entry for numeric non-config dir")
	ctxMap := skip.ContextMap()
	assert.Equal(t, nonConfig, ctxMap["path"])
	assert.NotEmpty(t, ctxMap["reason"], "skip log entry must include a reason")
}

// TestCleanupLegacyTempDirs_LogsFailure asserts a warn-level log entry with
// `path` and `error` fields is emitted when removal fails.
func TestCleanupLegacyTempDirs_LogsFailure(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("permission-based failure cannot be simulated when running as root")
	}

	ctx, logs := observedLoggerCtx(t)
	root := t.TempDir()

	legacy := filepath.Join(root, TempFilesFolder+"99")
	require.NoError(t, os.Mkdir(legacy, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(legacy, "fragment.yaml"), []byte("x: 1\n"), 0o600))
	require.NoError(t, os.Chmod(root, 0o500))
	t.Cleanup(func() { _ = os.Chmod(root, 0o700) })

	cleanupLegacyTempDirsIn(ctx, root)

	failLog := findLog(logs, "failed to remove legacy observe-agent temp directory")
	require.NotNil(t, failLog, "expected a failure log entry when removal is not permitted")
	ctxMap := failLog.ContextMap()
	assert.Equal(t, legacy, ctxMap["path"])
	assert.NotEmpty(t, ctxMap["error"], "failure log entry must include the underlying error")
	assert.GreaterOrEqual(t, failLog.Level, zapcore.WarnLevel,
		"failure log entry should be at Warn level or higher")
}

// findOverrideFragment returns the rendered otel_config_overrides fragment
// from a fragment list, or nil if none is present.
func findOverrideFragment(fragments []RenderedConfigFragment) *RenderedConfigFragment {
	const name = "otel-config-overrides.yaml"
	for i := range fragments {
		if fragments[i].Name == name {
			return &fragments[i]
		}
	}
	return nil
}

// TestSetupAndGetConfigs_OtelOverridePaths covers the viper delivery paths
// for otel_config_overrides: native map via viper.Set, JSON string env var
// (auto-coerced by GetStringMap), YAML string env var (falls through to
// yaml.Unmarshal), empty env var (ignored), and invalid YAML (error).
func TestSetupAndGetConfigs_OtelOverridePaths(t *testing.T) {
	want := map[string]any{
		"exporters": map[string]any{
			"otlp": map[string]any{
				"endpoint": "example.com:4317",
			},
		},
	}

	type setMode int
	const (
		setUnset setMode = iota
		setMap           // inject via viper.Set (native-map path)
		setEnv           // inject via env var + viper.AutomaticEnv
	)

	cases := []struct {
		name          string
		mode          setMode
		mapValue      map[string]any
		envValue      string
		wantFragment  bool
		wantOverrides map[string]any
		wantErrSubstr string
	}{
		{
			name:         "unset produces no override fragment",
			mode:         setUnset,
			wantFragment: false,
		},
		{
			name:          "native map from config is used directly",
			mode:          setMap,
			mapValue:      map[string]any{"exporters": map[string]any{"otlp": map[string]any{"endpoint": "example.com:4317"}}},
			wantFragment:  true,
			wantOverrides: want,
		},
		{
			name:          "JSON string env-var is coerced by GetStringMap",
			mode:          setEnv,
			envValue:      `{"exporters":{"otlp":{"endpoint":"example.com:4317"}}}`,
			wantFragment:  true,
			wantOverrides: want,
		},
		{
			name:          "YAML string env-var falls through to yaml.Unmarshal",
			mode:          setEnv,
			envValue:      "exporters:\n  otlp:\n    endpoint: example.com:4317\n",
			wantFragment:  true,
			wantOverrides: want,
		},
		{
			name:         "empty env-var is ignored",
			mode:         setEnv,
			envValue:     "",
			wantFragment: false,
		},
		{
			name:          "invalid YAML env-var returns parse error",
			mode:          setEnv,
			envValue:      "exporters:\n  otlp: [unterminated",
			wantErrSubstr: OTEL_OVERRIDE_YAML_KEY + " was provided but could not be parsed",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			setupTestViper(t)
			switch c.mode {
			case setMap:
				viper.Set(OTEL_OVERRIDE_YAML_KEY, c.mapValue)
			case setEnv:
				// AutomaticEnv + OTEL_CONFIG_OVERRIDES is the production
				// env-var path; unlike viper.Set it does not feed Unmarshal.
				t.Setenv("OTEL_CONFIG_OVERRIDES", c.envValue)
				viper.AutomaticEnv()
			}

			ctx := logger.WithCtx(context.Background(), logger.GetNop())
			fragments, err := SetupAndGetConfigs(ctx)

			if c.wantErrSubstr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), c.wantErrSubstr)
				return
			}
			require.NoError(t, err)

			got := findOverrideFragment(fragments)
			if !c.wantFragment {
				assert.Nil(t, got, "expected no override fragment")
				return
			}
			require.NotNil(t, got, "expected override fragment to be present")

			var decoded map[string]any
			require.NoError(t, yaml.Unmarshal([]byte(got.Content), &decoded),
				"override fragment content must be valid YAML")
			assert.Equal(t, c.wantOverrides, decoded)
		})
	}
}
