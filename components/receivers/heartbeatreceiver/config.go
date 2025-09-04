package heartbeatreceiver

import (
	"fmt"
	"time"
)

var validEnvironments = map[string]bool{
	"linux":      true,
	"macos":      true,
	"windows":    true,
	"kubernetes": true,
}

type Config struct {
	Interval    string          `mapstructure:"interval"`
	Environment string          `mapstructure:"environment"`
	AuthCheck   AuthCheckConfig `mapstructure:"auth_check"`
}

type AuthCheckConfig struct {
	URL     string           `mapstructure:"url"`
	Headers AuthCheckHeaders `mapstructure:"headers"`
}

type AuthCheckHeaders struct {
	Authorization string `mapstructure:"authorization"`
}

func (cfg *Config) Validate() error {
	interval, _ := time.ParseDuration(cfg.Interval)
	if interval.Seconds() < 5 {
		return fmt.Errorf("when defined, the interval has to be set to at least 1 minute (1m)")
	}

	// Validate environment field is required
	if cfg.Environment == "" {
		return fmt.Errorf("environment is required and must be one of: linux, macos, windows, kubernetes")
	}

	if !validEnvironments[cfg.Environment] {
		return fmt.Errorf("environment must be one of: linux, macos, windows, kubernetes")
	}

	return nil
}
