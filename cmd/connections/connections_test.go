package connections

import (
	"context"
	logger "observe-agent/cmd/commands/util"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/viper"
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

func (suite *ConnectionsTestSuite) MakeConnectionType(configFields []CollectorConfigFragment, v *viper.Viper) *ConnectionType {
	return MakeConnectionType("test", configFields, SelfMonitoringConnectionTypeName, WithConfigFolderPath(suite.configFilesPath), WithGetConfig(func() *viper.Viper { return v }))
}

func TestConnectionsTestSuite(t *testing.T) {
	suite.Run(t, new(ConnectionsTestSuite))
}

func (suite *ConnectionsTestSuite) TestConnections_RenderConfigTemplate() {
	ct := suite.MakeConnectionType([]CollectorConfigFragment{
		{configYAMLPath: "field1", colConfigFilePath: "config1.tpl"},
	}, nil)

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
	// Mock viper configuration
	v := viper.New()
	v.Set("field1", true)
	v.Set("field2", false)
	v.Set("field3", true)

	ct := suite.MakeConnectionType([]CollectorConfigFragment{
		{configYAMLPath: "field1", colConfigFilePath: "testConfig1.tpl"},
		{configYAMLPath: "field2", colConfigFilePath: "testConfig2.tpl"},
		{configYAMLPath: "field3", colConfigFilePath: ""},
	}, v)

	confValues := struct{}{}
	paths, err := ct.ProcessConfigFields(suite.ctx, suite.tempDir, v, confValues)
	suite.NoError(err)

	suite.Len(paths, 1)
	tmpFile := paths[0]
	tmpConfName := tmpFile[strings.LastIndex(tmpFile, "-")+1:]
	suite.Equal(ct.ConfigFields[0].colConfigFilePath, tmpConfName)
}

func (suite *ConnectionsTestSuite) TestConnectionType_GetConfigFilePaths() {
	// Mock viper configuration
	v := viper.New()
	v.Set("enabled", true)
	v.Set("field1", true)
	v.Set("field2", false)
	v.Set("field3", false)

	ct := suite.MakeConnectionType([]CollectorConfigFragment{
		{configYAMLPath: "field1", colConfigFilePath: "testConfig1.tpl"},
		{configYAMLPath: "field2", colConfigFilePath: "testConfig2.tpl"},
		{configYAMLPath: "field3", colConfigFilePath: ""},
	}, v)

	paths, err := ct.GetConfigFilePaths(suite.ctx, suite.tempDir)
	suite.NoError(err)
	suite.Len(paths, 1)
	tmpFile := paths[0]
	tmpConfName := tmpFile[strings.LastIndex(tmpFile, "-")+1:]
	suite.Equal(ct.ConfigFields[0].colConfigFilePath, tmpConfName)

	// Does nothing if not enabled
	v.Set("enabled", false)
	paths, err = ct.GetConfigFilePaths(suite.ctx, suite.tempDir)
	suite.NoError(err)
	suite.Len(paths, 0)
}
