package config

import (
	"context"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap"
)

// Loads the full snapshot's jmx receiver with the agent's own factories, so CI fails if the agent
// stops including the receiver or the receiver stops accepting the config. Validate isn't run: it
// hashes the JMX jar at jar_path, which CI doesn't have.
func Test_FullSnapshotJMXReceiver(t *testing.T) {
	setupConfig(t, snapshotTest{agentConfigPath: "test/snap1-full-agent-config.yaml", packageType: Linux})
	ctx := logger.WithCtx(context.Background(), logger.GetNop())
	settings, err := observecol.GetOtelCollectorSettings(ctx)
	require.NoError(t, err)

	factories, err := settings.Factories()
	require.NoError(t, err)
	factory, ok := factories.Receivers[component.MustNewType("jmx")]
	require.True(t, ok, "the agent's components don't include the jmx receiver")

	resolver, err := confmap.NewResolver(settings.ConfigProviderSettings.ResolverSettings)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, resolver.Shutdown(ctx)) })
	conf, err := resolver.Resolve(ctx)
	require.NoError(t, err)
	require.Contains(t, conf.Get("service::pipelines::metrics/jmx::receivers"), "jmx")

	jmxConf, err := conf.Sub("receivers::jmx")
	require.NoError(t, err)
	require.NoError(t, jmxConf.Unmarshal(factory.CreateDefaultConfig()))
}
