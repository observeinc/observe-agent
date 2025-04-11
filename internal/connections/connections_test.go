package connections

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/stretchr/testify/suite"
)

type ConnectionsTestSuite struct {
	suite.Suite
	tempDir         string
	configFilesPath string
	ctx             context.Context
}

func (suite *ConnectionsTestSuite) SetupSuite() {
	suite.ctx = logger.WithCtx(context.Background(), logger.Get())

	tempDir, err := os.MkdirTemp("", "test-connections")
	suite.NoError(err)
	suite.tempDir = tempDir

	_, filename, _, ok := runtime.Caller(0)
	suite.True(ok)
	suite.configFilesPath = path.Dir(filename)
}

func (suite *ConnectionsTestSuite) TearDownSuite() {
	os.RemoveAll(suite.tempDir)
}

var alwaysEnabled EnabledCheckFn = func(_ *config.AgentConfig) bool { return true }

func (suite *ConnectionsTestSuite) MakeConnectionType(configFields []CollectorConfigFragment, enableCheck EnabledCheckFn) *ConnectionType {
	return MakeConnectionType(
		"test",
		enableCheck,
		configFields,
		WithConfigFolderPath(suite.configFilesPath))
}

func TestConnectionsTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionsTestSuite))
}

func (suite *ConnectionsTestSuite) TestConnections_RenderConfigTemplate() {
	ct := suite.MakeConnectionType([]CollectorConfigFragment{
		{enabledCheck: alwaysEnabled, colConfigFilePath: "config1.tpl"},
	}, alwaysEnabled)

	// Test the RenderConfigTemplate function
	confValues := struct {
		TestStr  string
		TestArr1 []int
		TestArr2 []int
		TestObj  any
	}{
		TestStr:  "hello world",
		TestArr1: []int{1, 2, 3},
		TestArr2: []int{4, 5, 6},
		TestObj: struct {
			A string
			B int
			C []string
		}{
			A: "test",
			B: 7,
			C: []string{"test1", "test2", "test3"},
		},
	}
	result, err := ct.RenderConfigTemplate(suite.ctx, suite.tempDir, "testHelloWorld.tpl", confValues)

	suite.NoError(err)
	suite.NotEmpty(result)

	// Read the rendered content
	renderedContent, err := os.ReadFile(result)
	suite.NoError(err)
	expectedContent, err := os.ReadFile(filepath.Join(suite.configFilesPath, "test", "testHelloWorld.yaml"))
	suite.NoError(err)
	suite.Equal(string(expectedContent), string(renderedContent))
}

func (suite *ConnectionsTestSuite) TestConnectionType_ProcessConfigFields() {
	var agentConfig config.AgentConfig
	agentConfig.HostMonitoring.Enabled = true
	agentConfig.SelfMonitoring.Enabled = false
	agentConfig.Forwarding.Enabled = true

	ct := suite.MakeConnectionType([]CollectorConfigFragment{
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.HostMonitoring.Enabled }, colConfigFilePath: "testConfig1.tpl"},
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.SelfMonitoring.Enabled }, colConfigFilePath: "testConfig2.tpl"},
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.Forwarding.Enabled }, colConfigFilePath: ""},
	}, alwaysEnabled)

	paths, err := ct.ProcessConfigFields(suite.ctx, suite.tempDir, &agentConfig)
	suite.NoError(err)

	suite.Len(paths, 1)
	tmpFile := paths[0]
	tmpConfName := tmpFile[strings.LastIndex(tmpFile, "-")+1:]
	suite.Equal(ct.ConfigFields[0].colConfigFilePath, tmpConfName)
}

func (suite *ConnectionsTestSuite) TestConnectionType_GetConfigFilePaths() {
	var agentConfig config.AgentConfig
	agentConfig.Debug = true
	agentConfig.HostMonitoring.Enabled = true
	agentConfig.SelfMonitoring.Enabled = false
	agentConfig.Forwarding.Enabled = true

	ct := suite.MakeConnectionType([]CollectorConfigFragment{
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.HostMonitoring.Enabled }, colConfigFilePath: "testConfig1.tpl"},
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.SelfMonitoring.Enabled }, colConfigFilePath: "testConfig2.tpl"},
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.Forwarding.Enabled }, colConfigFilePath: ""},
	}, func(ac *config.AgentConfig) bool { return ac.Debug })

	paths, err := ct.GetConfigFilePaths(suite.ctx, suite.tempDir, &agentConfig)
	suite.NoError(err)
	suite.Len(paths, 1)
	tmpFile := paths[0]
	tmpConfName := tmpFile[strings.LastIndex(tmpFile, "-")+1:]
	suite.Equal(ct.ConfigFields[0].colConfigFilePath, tmpConfName)

	// Does nothing if not enabled
	agentConfig.Debug = false
	paths, err = ct.GetConfigFilePaths(suite.ctx, suite.tempDir, &agentConfig)
	suite.NoError(err)
	suite.Len(paths, 0)
}
