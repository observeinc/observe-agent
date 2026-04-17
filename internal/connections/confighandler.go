package connections

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	OTEL_OVERRIDE_YAML_KEY = "otel_config_overrides"
)

// SetupAndGetConfigs renders every enabled otel config fragment in memory
// and returns them. It also sweeps any legacy on-disk config temp dirs left
// behind by older agent versions. See CleanupLegacyTempDirs.
func SetupAndGetConfigs(ctx context.Context) ([]RenderedConfigFragment, error) {
	CleanupLegacyTempDirs(ctx)

	agentConfig, err := config.AgentConfigFromViper(viper.GetViper())
	if err != nil {
		return nil, err
	}

	fragments := make([]RenderedConfigFragment, 0)
	for _, conn := range AllConnectionTypes {
		connFragments, err := conn.GetBundledConfigs(ctx, agentConfig)
		if err != nil {
			return nil, err
		}
		fragments = append(fragments, connFragments...)
	}

	if viper.IsSet(OTEL_OVERRIDE_YAML_KEY) {
		// GetStringMap is more lenient with respect to conversions than Sub, which only handles maps.
		overrides := viper.GetStringMap(OTEL_OVERRIDE_YAML_KEY)
		if len(overrides) == 0 {
			stringData := viper.GetString(OTEL_OVERRIDE_YAML_KEY)
			// If this was truly set to empty, then ignore it.
			if stringData != "" {
				// Viper can handle overrides set in the agent config, or passed in as an env var as a JSON string.
				// For consistency, we also want to accept an env var as a YAML string.
				if err := yaml.Unmarshal([]byte(stringData), &overrides); err != nil {
					return nil, fmt.Errorf("%s was provided but could not be parsed", OTEL_OVERRIDE_YAML_KEY)
				}
			}
		}
		// Only build an override fragment if there are overrides present
		// (ie ignore empty maps).
		if len(overrides) != 0 {
			override, err := buildOverrideConfigFragment(overrides)
			if err != nil {
				return nil, err
			}
			fragments = append(fragments, override)
		}
	}

	if l := logger.FromCtx(ctx); l != nil {
		names := make([]string, 0, len(fragments))
		for _, f := range fragments {
			names = append(names, f.Name)
		}
		l.Debug("rendered otel config fragments", zap.Strings("fragments", names))
	}
	return fragments, nil
}

func buildOverrideConfigFragment(data map[string]any) (RenderedConfigFragment, error) {
	contents, err := yaml.Marshal(data)
	if err != nil {
		return RenderedConfigFragment{}, fmt.Errorf("failed to marshal otel config overrides: %w", err)
	}
	return RenderedConfigFragment{Name: "otel-config-overrides.yaml", Content: string(contents)}, nil
}

func isLegacyTempDirName(name string) bool {
	if !strings.HasPrefix(name, TempFilesFolder) {
		return false
	}
	suffix := strings.TrimPrefix(name, TempFilesFolder)
	if suffix == "" {
		return false
	}
	for _, r := range suffix {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// classifyLegacyConfigTempDir reports whether a candidate directory looks like
// a legacy config temp dir, and if not, returns a human-readable reason
// suitable for debug logging.
func classifyLegacyConfigTempDir(path string) (match bool, reason string) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, fmt.Sprintf("cannot read directory: %s", err.Error())
	}
	// Empty dirs match: the flat-and-YAML-only rule is vacuously true.
	if len(entries) == 0 {
		return true, ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return false, fmt.Sprintf("contains subdirectory %q (legacy config dirs were flat)", entry.Name())
		}
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			return false, fmt.Sprintf("contains non-YAML file %q", entry.Name())
		}
	}
	return true, ""
}

// CleanupLegacyTempDirs removes directories in os.TempDir() whose names
// match `<TempFilesFolder><numeric-suffix>` and whose contents are either
// empty or only YAML files. The sweep is best-effort and non-fatal; all
// outcomes are emitted as structured logs.
func CleanupLegacyTempDirs(ctx context.Context) {
	cleanupLegacyTempDirsIn(ctx, os.TempDir())
}

func cleanupLegacyTempDirsIn(ctx context.Context, root string) {
	log := logger.FromCtx(ctx)
	log.Debug("scanning temp directory for legacy observe-agent config directories",
		zap.String("temp_dir", root))

	entries, err := os.ReadDir(root)
	if err != nil {
		log.Warn("failed to scan temp directory for legacy observe-agent config directories",
			zap.String("temp_dir", root), zap.Error(err))
		return
	}

	var scanned, matched, removed, failed, skipped int
	for _, entry := range entries {
		scanned++
		if !entry.IsDir() || !isLegacyTempDirName(entry.Name()) {
			continue
		}
		matched++
		p := filepath.Join(root, entry.Name())

		ok, reason := classifyLegacyConfigTempDir(p)
		if !ok {
			skipped++
			log.Debug("skipping candidate temp directory",
				zap.String("path", p), zap.String("reason", reason))
			continue
		}

		if err := os.RemoveAll(p); err != nil {
			failed++
			log.Warn("failed to remove legacy observe-agent temp directory",
				zap.String("path", p), zap.Error(err))
			continue
		}
		removed++
		log.Info("removed legacy observe-agent temp directory", zap.String("path", p))
	}

	fields := []zap.Field{
		zap.String("temp_dir", root),
		zap.Int("scanned", scanned),
		zap.Int("matched", matched),
		zap.Int("removed", removed),
		zap.Int("failed", failed),
		zap.Int("skipped", skipped),
	}
	const summaryMsg = "legacy observe-agent temp directory cleanup finished"
	switch {
	case failed > 0:
		log.Warn(summaryMsg, fields...)
	case removed > 0:
		log.Info(summaryMsg, fields...)
	default:
		log.Debug(summaryMsg, fields...)
	}
}
