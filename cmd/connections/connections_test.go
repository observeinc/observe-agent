package connections

var MockConnectionType = ConnectionType{
	Name: "test",
	ConfigFields: []CollectorConfigFragment{
		{configYAMLPath: "testField1", colConfigFilePath: "testConfig1.tpl"},
		// {configYAMLPath: "field2", colConfigFilePath: "config2.tpl"},
	},
	Type: "test_type",
}

// 'func TestConnections_RenderConfigTemplate(t *testing.T) {
// 	// Create a temporary directory for test files
// 	tempDir, err := os.MkdirTemp("", "test-connections")
// 	assert.NoError(t, err)
// 	defer os.RemoveAll(tempDir)
// 	originalOsTempDir := os.TempDir
// 	os.TempDir = func() string { return tempDir }
// 	defer func() { os.Tempdir = originalOsTempDir }()

// 	// Mock the GetConfigFolderPath function
// 	originalGetConfigFolderPath := GetConfigFolderPath
// 	GetConfigFolderPath = func() string { return "./" }
// 	defer func() { GetConfigFolderPath = originalGetConfigFolderPath }()

// 	// Test the RenderConfigTemplate function
// 	ctx := context.Background()
// 	confValues := struct{ Name string }{"World"}
// 	result, err := ct.RenderConfigTemplate(ctx, "test.tpl", confValues)

// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, result)

// 	// Read the rendered content
// 	renderedContent, err := os.ReadFile(result)
// 	assert.NoError(t, err)
// 	assert.Equal(t, "Hello, World!", string(renderedContent))
// }

// func TestConnectionType_ProcessConfigFields(t *testing.T) {
// 	ct := &ConnectionType{
// 		Name: "test",
// 		ConfigFields: []CollectorConfigFragment{
// 			{configYAMLPath: "field1", colConfigFilePath: "config1.tpl"},
// 			{configYAMLPath: "field2", colConfigFilePath: "config2.tpl"},
// 			{configYAMLPath: "field3", colConfigFilePath: ""},
// 		},
// 	}

// 	// Mock viper configuration
// 	v := viper.New()
// 	v.Set("field1", true)
// 	v.Set("field2", false)
// 	v.Set("field3", true)

// 	// Mock RenderConfigTemplate
// 	originalRenderConfigTemplate := ct.RenderConfigTemplate
// 	ct.RenderConfigTemplate = func(ctx context.Context, tplFilename string, confValues any) (string, error) {
// 		return "/tmp/" + tplFilename, nil
// 	}
// 	defer func() { ct.RenderConfigTemplate = originalRenderConfigTemplate }()

// 	ctx := context.Background()
// 	confValues := struct{}{}
// 	paths, err := ct.ProcessConfigFields(ctx, v, confValues)

// 	assert.NoError(t, err)
// 	assert.Equal(t, []string{"/tmp/config1.tpl"}, paths)
// }

// func TestConnectionType_GetConfigFilePaths(t *testing.T) {
// 	ct := &ConnectionType{
// 		Name: "test",
// 		Type: "self_monitoring",
// 		ConfigFields: []CollectorConfigFragment{
// 			{configYAMLPath: "field1", colConfigFilePath: "config1.tpl"},
// 			{configYAMLPath: "field2", colConfigFilePath: "config2.tpl"},
// 		},
// 	}

// 	// Mock viper configuration
// 	v := viper.New()
// 	v.Set("test.enabled", true)
// 	v.Set("test.field1", true)
// 	v.Set("test.field2", false)

// 	// Mock ProcessConfigFields
// 	originalProcessConfigFields := ct.ProcessConfigFields
// 	ct.ProcessConfigFields = func(ctx context.Context, rawConnConfig *viper.Viper, confValues any) ([]string, error) {
// 		return []string{"/tmp/config1.tpl"}, nil
// 	}
// 	defer func() { ct.ProcessConfigFields = originalProcessConfigFields }()

// 	// Set viper.GetViper() to return our mock configuration
// 	originalGetViper := viper.GetViper
// 	viper.GetViper = func() *viper.Viper { return v }
// 	defer func() { viper.GetViper = originalGetViper }()

// 	ctx := context.Background()
// 	paths := ct.GetConfigFilePaths(ctx)

// 	assert.Equal(t, []string{"/tmp/config1.tpl"}, paths)
// }
