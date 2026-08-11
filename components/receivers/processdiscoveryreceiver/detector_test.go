package processdiscoveryreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeDetector(t *testing.T) {
	detector := newRuntimeDetector(supportedRuntimes)
	tests := []struct {
		command string
		args    []string
		runtime string
	}{
		{"java", []string{"java", "Main"}, "java"},
		{"python3.12", []string{"python3.12", "app.py"}, "python"},
		{"node", []string{"node", "app.js"}, "nodejs"},
		{"dotnet", []string{"dotnet", "app.dll"}, "dotnet"},
		{"beam.smp", []string{"beam.smp", "-elixir_root", "/opt/elixir"}, "elixir"},
		{"php-fpm", []string{"php-fpm"}, "php"},
	}
	for _, test := range tests {
		result, ok := detector.detectCommand(test.command, test.args)
		require.True(t, ok, test.command)
		assert.Equal(t, test.runtime, result.runtimeFamily)
		assert.NotEmpty(t, result.assertions)
	}
}

func TestExecutableSuffixIsInferredVersion(t *testing.T) {
	result, ok := newRuntimeDetector(supportedRuntimes).detectCommand("python3.12", nil)
	require.True(t, ok)
	assert.Empty(t, result.runtimeVersion)
	assert.Equal(t, "3.12", result.inferredVersion)
	assert.Equal(t, "executable_suffix", result.inferredVersionKind)
}

func TestUnknownCommand(t *testing.T) {
	_, ok := newRuntimeDetector(supportedRuntimes).detectCommand("native-app", nil)
	assert.False(t, ok)
}

func TestExecutableNameCanEnrichInterpreterCommand(t *testing.T) {
	detector := newRuntimeDetector(supportedRuntimes)
	commandResult, commandFound := detector.detectCommand("python", []string{"python", "app.py"})
	executableResult, executableFound := detector.detectCommand("python3.12", []string{"python", "app.py"})
	require.True(t, commandFound)
	require.True(t, executableFound)
	mergeDetectionResult(&commandResult, &commandFound, executableResult)
	assert.Equal(t, "3.12", commandResult.inferredVersion)
	assert.Equal(t, "executable_suffix", commandResult.inferredVersionKind)
}

func TestConflictingRuntimeEvidenceIsNotMerged(t *testing.T) {
	result := detectionResult{runtimeFamily: "java", assertions: []string{"java.executable"}}
	found := true
	mergeDetectionResult(&result, &found, detectionResult{
		runtimeFamily: "python", inferredVersion: "3.12", inferredVersionKind: "executable_suffix",
		assertions: []string{"python.executable"},
	})
	assert.Equal(t, "java", result.runtimeFamily)
	assert.Empty(t, result.inferredVersion)
	assert.Equal(t, []string{"java.executable"}, result.assertions)
}

func TestNativeEvidenceCanSupportIdentifiedRuntime(t *testing.T) {
	result := detectionResult{runtimeFamily: "java", assertions: []string{"java.executable"}}
	found := true
	mergeDetectionResult(&result, &found, detectionResult{
		runtimeFamily: "native", assertions: []string{"native.libstdcxx_loaded"},
	})
	assert.Equal(t, "java", result.runtimeFamily)
	assert.Equal(t, []string{"java.executable", "native.libstdcxx_loaded"}, result.assertions)
}
