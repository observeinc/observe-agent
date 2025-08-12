package heartbeatreceiver

import (
	"fmt"
	"time"
)

type Config struct {
	Interval string `mapstructure:"interval"`
}

func (cfg *Config) Validate() error {
	interval, _ := time.ParseDuration(cfg.Interval)
	if interval.Minutes() < 1 {
		return fmt.Errorf("when defined, the interval has to be set to at least 1 minute (1m)")
	}
	return nil
}
