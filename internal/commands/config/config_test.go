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
	"strings"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/connections"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TODO rework this test to handle our snapshot tests as go unit tests.
func Test_RenderOtelConfig(t *testing.T) {
	// Get current path
	_, filename, _, ok := runtime.Caller(0)
	assert.True(t, ok)
	curPath := path.Dir(filename)

	// Set the template base dir for all connections
	for _, conn := range connections.AllConnectionTypes {
		conn.ApplyOptions(connections.WithConfigFolderPath(filepath.Join(curPath, "../../../packaging/macos/connections")))
	}

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
	assert.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(output.String()))
}
