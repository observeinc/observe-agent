package processdiscoveryreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestDefaultConfigValid(t *testing.T) {
	cfg := createDefaultConfig().(*Config)
	require.NoError(t, cfg.Validate())
}

func TestConfigValidation(t *testing.T) {
	tests := map[string]func(*Config){
		"short interval":               func(c *Config) { c.CollectionInterval = time.Second },
		"negative delay":               func(c *Config) { c.InitialDelay = -time.Second },
		"zero grace":                   func(c *Config) { c.TerminationGraceScans = 0 },
		"zero tracked":                 func(c *Config) { c.MaxTrackedProcesses = 0 },
		"relative procfs":              func(c *Config) { c.ProcFSPath = "proc" },
		"relative machine id":          func(c *Config) { c.HostMachineIDPath = "etc/machine-id" },
		"empty runtimes":               func(c *Config) { c.Runtimes = nil },
		"unsupported runtime":          func(c *Config) { c.Runtimes = []string{"brainfuck"} },
		"duplicate runtime":            func(c *Config) { c.Runtimes = []string{"java", "java"} },
		"invalid otlp max connections": func(c *Config) { c.OTLPDetection.MaxConnectionsPerProcess = 0 },
		"oversized commandline":        func(c *Config) { c.MaxCmdlineBytes = maxCmdlineBytes + 1 },
		"zero cache ttl":               func(c *Config) { c.RuntimeDetection.BinaryCacheTTL = 0 },
		"zero cache size":              func(c *Config) { c.RuntimeDetection.BinaryCacheSize = 0 },
		"zero lifecycle buffer":        func(c *Config) { c.Lifecycle.BufferSize = 0 },
		"zero min lifetime scans":      func(c *Config) { c.Filtering.MinLifetimeScans = 0 },
		"invalid include path glob":    func(c *Config) { c.Filtering.Include.ExecutablePaths = []string{"[invalid"} },
		"invalid exclude path glob":    func(c *Config) { c.Filtering.Exclude.ExecutablePaths = []string{"[invalid"} },
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			cfg := createDefaultConfig().(*Config)
			mutate(cfg)
			require.Error(t, cfg.Validate())
		})
	}
}
