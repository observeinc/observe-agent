package servicediscovery

import (
	"fmt"
	"time"
)

// Config represents the connector config settings within the collector's config.yaml
type Config struct {
	// LogExportInterval is the interval at which to export OTLP logs for all services
	// Default: 1 hour
	LogExportInterval time.Duration `mapstructure:"log_export_interval"`
}

func (c *Config) Validate() error {
	if c.LogExportInterval <= 0 {
		return fmt.Errorf("log_export_interval must be positive, got %v", c.LogExportInterval)
	}
	return nil
}
