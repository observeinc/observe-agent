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
	"docker":     true,
}

type Config struct {
	Interval       string          `mapstructure:"interval"`
	ConfigInterval string          `mapstructure:"config_interval"`
	Environment    string          `mapstructure:"environment"`
	AuthCheck      AuthCheckConfig `mapstructure:"auth_check"`
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
	if interval.Hours() > 8 {
		return fmt.Errorf(("when defined, the interval must be set to a maximum of 8 hours (8h)"))
	}

	// Validate config heartbeat interval if set
	if cfg.ConfigInterval != "" {
		configInterval, err := time.ParseDuration(cfg.ConfigInterval)
		if err != nil {
			return fmt.Errorf("invalid config_interval: %w", err)
		}
		if configInterval.Minutes() < 10 {
			return fmt.Errorf("config_interval must be at least 10 minutes")
		}
		if configInterval.Hours() > 24 {
			return fmt.Errorf("config_interval must be at most 24 hours")
		}
	}

	// Validate environment field is required
	if cfg.Environment == "" {
		return fmt.Errorf("environment is required and must be one of: linux, macos, windows, docker, kubernetes")
	}

	if !validEnvironments[cfg.Environment] {
		return fmt.Errorf("environment must be one of: linux, macos, windows, docker, kubernetes")
	}

	return nil
}
