/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"bytes"
	"context"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TODO enable this test once the agent bundled configs don't depend on the host filesystem having the templates.
func XTest_RenderOtelConfig(t *testing.T) {
	// Get current path
	_, filename, _, ok := runtime.Caller(0)
	assert.True(t, ok)
	curPath := path.Dir(filename)

	// Set config flags
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	observecol.AddConfigFlags(flags)
	flags.Parse([]string{"--config", filepath.Join(curPath, "test/otel-config.yaml")})
	viper.Reset()
	root.CfgFile = filepath.Join(curPath, "test/agent-config.yaml")
	root.InitConfig()

	// Run the test
	ctx := logger.WithCtx(context.Background(), logger.GetNop())
	var output bytes.Buffer
	PrintShortOtelConfig(ctx, &output)
	expected, err := os.ReadFile(filepath.Join(curPath, "test/output.yaml"))
	assert.NoError(t, err)
	assert.Equal(t, string(expected), output.String())
}
