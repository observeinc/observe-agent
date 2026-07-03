package servicediscovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConfigValidate tests the config validation
func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config with 1 hour",
			config: &Config{
				LogExportInterval: 1 * time.Hour,
			},
			expectError: false,
		},
		{
			name: "valid config with 5 minutes",
			config: &Config{
				LogExportInterval: 5 * time.Minute,
			},
			expectError: false,
		},
		{
			name: "invalid config with zero interval",
			config: &Config{
				LogExportInterval: 0,
			},
			expectError: true,
			errorMsg:    "log_export_interval must be positive",
		},
		{
			name: "invalid config with negative interval",
			config: &Config{
				LogExportInterval: -1 * time.Hour,
			},
			expectError: true,
			errorMsg:    "log_export_interval must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDefaultConfig tests that the default config has sensible values
func TestDefaultConfig(t *testing.T) {
	cfg := createDefaultConfig().(*Config)

	assert.Equal(t, 1*time.Minute, cfg.LogExportInterval)
	assert.NoError(t, cfg.Validate())
}

