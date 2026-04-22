package connections

import (
	"context"
	"embed"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/stretchr/testify/suite"
)

var (
	//go:embed test/testConfig1.tpl
	testConfig1FS embed.FS
	//go:embed test/testHelloWorld.tpl
	testHelloWorldFS embed.FS
)

var TestTemplateOverrides = map[string]embed.FS{
	"test/testConfig1.tpl":    testConfig1FS,
	"test/testHelloWorld.tpl": testHelloWorldFS,
}

type ConnectionsTestSuite struct {
	suite.Suite
	configFilesPath string
	ctx             context.Context
}

func (suite *ConnectionsTestSuite) SetupSuite() {
	suite.ctx = logger.WithCtx(context.Background(), logger.Get())

	_, filename, _, ok := runtime.Caller(0)
	suite.True(ok)
	suite.configFilesPath = path.Dir(filename)
}

func (suite *ConnectionsTestSuite) MakeConnectionType(configFields []BundledConfigFragment, enableCheck EnabledCheckFn) *ConnectionType {
	return MakeConnectionType(
		"test",
		enableCheck,
		configFields,
		WithConfigTemplateOverrides(TestTemplateOverrides),
	)
}

func TestConnectionsTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionsTestSuite))
}

func (suite *ConnectionsTestSuite) TestConnections_RenderConfigTemplate() {
	ct := suite.MakeConnectionType([]BundledConfigFragment{
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
	result, err := ct.renderBundledConfigTemplate(suite.ctx, "testHelloWorld.tpl", confValues)

	suite.NoError(err)
	suite.NotEmpty(result.Content)
	suite.Equal("test-testHelloWorld.tpl", result.Name)

	expectedContent, err := os.ReadFile(filepath.Join(suite.configFilesPath, "test", "testHelloWorld.yaml"))
	suite.NoError(err)
	suite.Equal(string(expectedContent), result.Content)
}

func (suite *ConnectionsTestSuite) TestConnectionType_ProcessConfigFields() {
	var agentConfig config.AgentConfig
	agentConfig.HostMonitoring.Enabled = true
	agentConfig.SelfMonitoring.Enabled = false
	agentConfig.Forwarding.Enabled = true

	ct := suite.MakeConnectionType([]BundledConfigFragment{
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.HostMonitoring.Enabled }, colConfigFilePath: "testConfig1.tpl"},
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.SelfMonitoring.Enabled }, colConfigFilePath: "testConfig2.tpl"},
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.Forwarding.Enabled }, colConfigFilePath: ""},
	}, alwaysEnabled)

	fragments, err := ct.renderAllBundledConfigFragments(suite.ctx, &agentConfig)
	suite.NoError(err)

	suite.Len(fragments, 1)
	suite.Equal("test-"+ct.BundledConfigFragments[0].colConfigFilePath, fragments[0].Name)
	suite.NotEmpty(fragments[0].Content)
}

func (suite *ConnectionsTestSuite) TestConnectionType_GetConfigFilePaths() {
	var agentConfig config.AgentConfig
	agentConfig.Debug = true
	agentConfig.HostMonitoring.Enabled = true
	agentConfig.SelfMonitoring.Enabled = false
	agentConfig.Forwarding.Enabled = true

	ct := suite.MakeConnectionType([]BundledConfigFragment{
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.HostMonitoring.Enabled }, colConfigFilePath: "testConfig1.tpl"},
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.SelfMonitoring.Enabled }, colConfigFilePath: "testConfig2.tpl"},
		{enabledCheck: func(ac *config.AgentConfig) bool { return ac.Forwarding.Enabled }, colConfigFilePath: ""},
	}, func(ac *config.AgentConfig) bool { return ac.Debug })

	fragments, err := ct.GetBundledConfigs(suite.ctx, &agentConfig)
	suite.NoError(err)
	suite.Len(fragments, 1)
	suite.Equal("test-"+ct.BundledConfigFragments[0].colConfigFilePath, fragments[0].Name)

	// Does nothing if not enabled
	agentConfig.Debug = false
	fragments, err = ct.GetBundledConfigs(suite.ctx, &agentConfig)
	suite.NoError(err)
	suite.Len(fragments, 0)
}
