/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"bytes"
	"context"
	"embed"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/connections"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

type PackageType string

const MacOS = PackageType("macos")
const Linux = PackageType("linux")
const Windows = PackageType("windows")
const Docker = PackageType("docker")

type snapshotTest struct {
	agentConfigPath string
	otelConfigPath  string
	outputPath      string
	packageType     PackageType
}

var allSnapshotTests = []snapshotTest{
	// Tests with default agent config
	{
		agentConfigPath: "test/snap0-default-agent-config.yaml",
		outputPath:      "test/snap0-docker-output.yaml",
		packageType:     Docker,
	},
	{
		agentConfigPath: "test/snap0-default-agent-config.yaml",
		outputPath:      "test/snap0-linux-output.yaml",
		packageType:     Linux,
	},
	{
		agentConfigPath: "test/snap0-default-agent-config.yaml",
		outputPath:      "test/snap0-macos-output.yaml",
		packageType:     MacOS,
	},
	{
		agentConfigPath: "test/snap0-default-agent-config.yaml",
		outputPath:      "test/snap0-windows-output.yaml",
		packageType:     Windows,
	},
	// Tests with full agent config
	{
		agentConfigPath: "test/snap1-full-agent-config.yaml",
		outputPath:      "test/snap1-docker-output.yaml",
		packageType:     Docker,
	},
	{
		agentConfigPath: "test/snap1-full-agent-config.yaml",
		outputPath:      "test/snap1-linux-output.yaml",
		packageType:     Linux,
	},
	{
		agentConfigPath: "test/snap1-full-agent-config.yaml",
		outputPath:      "test/snap1-macos-output.yaml",
		packageType:     MacOS,
	},
	{
		agentConfigPath: "test/snap1-full-agent-config.yaml",
		outputPath:      "test/snap1-windows-output.yaml",
		packageType:     Windows,
	},
	// Tests with minimal agent config
	{
		agentConfigPath: "test/snap2-empty-agent-config.yaml",
		otelConfigPath:  "test/snap2-otel-config.yaml",
		outputPath:      "test/snap2-with-otel-output.yaml",
		packageType:     MacOS,
	},
	{
		agentConfigPath: "test/snap2-empty-agent-config.yaml",
		outputPath:      "test/snap2-windows-output.yaml",
		packageType:     Windows,
	},
}

// This test cannot be run in our CI since some of the OTel components validate file paths
// which are not set up in our CI environment (ex checking that the file storage dir exists).
func XTest_ValidateOtelConfig(t *testing.T) {
	for _, test := range allSnapshotTests {
		// Skip environments that don't match the current OS; some OTel component behavior is OS-specific.
		switch test.packageType {
		case MacOS:
			if runtime.GOOS != "darwin" {
				continue
			}
		case Linux, Docker:
			if runtime.GOOS != "linux" {
				continue
			}
		case Windows:
			if runtime.GOOS != "windows" {
				continue
			}
		}
		t.Run(test.outputPath, func(t *testing.T) {
			runValidateTest(t, test)
		})
	}
}

func runValidateTest(t *testing.T, test snapshotTest) {
	setupConfig(t, test)

	// Run the test
	ctx := logger.WithCtx(context.Background(), logger.GetNop())
	col, cleanup, err := observecol.GetOtelCollector(ctx)
	if cleanup != nil {
		defer cleanup()
	}
	assert.NoError(t, err)
	err = col.DryRun(ctx)
	assert.NoError(t, err)
}

func Test_RenderOtelConfig(t *testing.T) {
	for _, test := range allSnapshotTests {
		t.Run(test.outputPath, func(t *testing.T) {
			runSnapshotTest(t, test)
		})
	}
}

func runSnapshotTest(t *testing.T, test snapshotTest) {
	setupConfig(t, test)

	// Run the test
	curPath := getCurPath()
	ctx := logger.WithCtx(context.Background(), logger.GetNop())
	var output bytes.Buffer
	err := PrintShortOtelConfig(ctx, &output)
	assert.NoError(t, err)
	expected, err := os.ReadFile(filepath.Join(curPath, test.outputPath))
	assert.NoError(t, err)
	assert.Equal(t, strings.TrimSpace(string(expected)), strings.TrimSpace(output.String()))
}

func setupConfig(t *testing.T, test snapshotTest) {
	// Set the template overrides for all connections
	for _, conn := range connections.AllConnectionTypes {
		conn.ApplyOptions(connections.WithConfigTemplateOverrides(getTemplateOverrides(t, test.packageType)))
	}

	// Set config flags
	curPath := getCurPath()
	flags := pflag.NewFlagSet("test", pflag.ContinueOnError)
	observecol.AddConfigFlags(flags)
	if test.otelConfigPath != "" {
		flags.Parse([]string{"--config", filepath.Join(curPath, test.otelConfigPath)})
	}
	viper.Reset()
	root.CfgFile = filepath.Join(curPath, test.agentConfigPath)
	root.InitConfig()
	setEnvVars(t, test.packageType)
}

func getTemplateOverrides(t *testing.T, packageType PackageType) map[string]embed.FS {
	switch packageType {
	case MacOS:
		return bundledconfig.MacOSTemplateFS
	case Linux:
		return bundledconfig.LinuxTemplateFS
	case Windows:
		return bundledconfig.WindowsTemplateFS
	case Docker:
		return bundledconfig.DockerTemplateFS
	default:
		t.Errorf("Unknown package type: %s", packageType)
		return nil
	}
}

func setEnvVars(t *testing.T, packageType PackageType) {
	os.Setenv("TEST_ENV_VAR", "test-value")
	// Set a predictable agent instance ID for tests
	assert.NoError(t, os.Setenv("OBSERVE_AGENT_INSTANCE_ID", "test-agent-instance-id"))

	switch packageType {
	case MacOS:
		assert.NoError(t, os.Setenv("FILESTORAGE_PATH", "/var/lib/observe-agent/filestorage"))
		assert.NoError(t, os.Setenv("OBSERVE_AGENT_CONFIG_PATH", "/usr/local/observe-agent/observe-agent.yaml"))
	case Windows:
		assert.NoError(t, os.Setenv("FILESTORAGE_PATH", "C:\\ProgramData\\Observe\\observe-agent\\filestorage"))
		assert.NoError(t, os.Setenv("OBSERVE_AGENT_CONFIG_PATH", "C:\\Program Files\\Observe\\observe-agent\\observe-agent.yaml"))
	case Linux, Docker:
		assert.NoError(t, os.Setenv("FILESTORAGE_PATH", "/var/lib/observe-agent/filestorage"))
		assert.NoError(t, os.Setenv("OBSERVE_AGENT_CONFIG_PATH", "/etc/observe-agent/observe-agent.yaml"))
	default:
		t.Errorf("Unknown package type: %s", packageType)
	}

}

func getCurPath() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("Failed to get current path")
	}
	return path.Dir(filename)
}
