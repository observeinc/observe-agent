package config

import (
	"context"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Upcoming change warnings are for user config, so the bundled configuration must never trigger one.
func Test_BundledConfigHasNoUpcomingChanges(t *testing.T) {
	require.NoError(t, observecol.ApplyFeatureGates(context.Background()))
	for _, test := range allSnapshotTests {
		t.Run(test.agentConfigPath+"/"+string(test.packageType), func(t *testing.T) {
			setupConfig(t, test)
			ctx := logger.WithCtx(context.Background(), logger.GetNop())
			settings, err := observecol.GetOtelCollectorSettings(ctx)
			require.NoError(t, err)
			changes, err := observecol.UpcomingChanges(ctx, settings)
			require.NoError(t, err)
			assert.Empty(t, changes)
		})
	}
}
