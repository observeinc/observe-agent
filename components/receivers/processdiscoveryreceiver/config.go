package processdiscoveryreceiver

import (
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"go.opentelemetry.io/collector/component"
)

const (
	defaultCollectionInterval    = 10 * time.Second
	defaultReportInterval        = 5 * time.Minute
	defaultInitialDelay          = time.Second
	defaultTerminationGraceScans = 2
	defaultMaxTrackedProcesses   = 10000
	defaultProcFSPath            = "/proc"
	defaultMaxCmdlineBytes       = 8192
	maxCmdlineBytes              = 64 * 1024
	defaultBinaryCacheTTL        = 5 * time.Minute
	defaultBinaryCacheSize       = 1024
	defaultLifecycleBufferSize   = 4096
	defaultClockTicks            = 100
	defaultMaxCommandArgs        = 128
)

var supportedRuntimes = []string{
	"java",
	"python",
	"nodejs",
	"dotnet",
	"ruby",
	"go",
	"rust",
	"erlang",
	"elixir",
	"php",
	"perl",
}

var defaultSensitiveWords = []string{
	"password", "passwd", "pwd", "access_token", "auth_token", "api_key", "apikey",
	"secret", "credentials", "client_secret", "private_key", "token",
}

type RuntimeDetectionConfig struct {
	BinaryCacheTTL  time.Duration `mapstructure:"binary_cache_ttl"`
	BinaryCacheSize int           `mapstructure:"binary_cache_size"`
}

type LifecycleConfig struct {
	Enabled    bool `mapstructure:"enabled"`
	BufferSize int  `mapstructure:"buffer_size"`
}

type CommandLineConfig struct {
	Enabled              bool     `mapstructure:"enabled"`
	StripAllArguments    bool     `mapstructure:"strip_all_arguments"`
	MaxArguments         int      `mapstructure:"max_arguments"`
	CustomSensitiveWords []string `mapstructure:"custom_sensitive_words"`
}

type NetworkConfig struct {
	Enabled            bool `mapstructure:"enabled"`
	MaxPortsPerProcess int  `mapstructure:"max_ports_per_process"`
	MaxSeries          int  `mapstructure:"max_series"`
}

type Config struct {
	CollectionInterval    time.Duration          `mapstructure:"collection_interval"`
	ReportInterval        time.Duration          `mapstructure:"report_interval"`
	InitialDelay          time.Duration          `mapstructure:"initial_delay"`
	TerminationGraceScans int                    `mapstructure:"termination_grace_scans"`
	MaxTrackedProcesses   int                    `mapstructure:"max_tracked_processes"`
	ProcFSPath            string                 `mapstructure:"procfs_path"`
	Runtimes              []string               `mapstructure:"runtimes"`
	MaxCmdlineBytes       int                    `mapstructure:"max_cmdline_bytes"`
	RuntimeDetection      RuntimeDetectionConfig `mapstructure:"runtime_detection"`
	Lifecycle             LifecycleConfig        `mapstructure:"lifecycle"`
	ClockTicks            uint64                 `mapstructure:"clock_ticks"`
	Kubernetes            KubernetesConfig       `mapstructure:"kubernetes"`
	HostID                string                 `mapstructure:"host_id"`
	HostArch              string                 `mapstructure:"host_arch"`
	HostMachineIDPath     string                 `mapstructure:"host_machine_id_path"`
	CommandLine           CommandLineConfig      `mapstructure:"command_line"`
	Network               NetworkConfig          `mapstructure:"network"`
}

func createDefaultConfig() component.Config {
	return &Config{
		CollectionInterval:    defaultCollectionInterval,
		ReportInterval:        defaultReportInterval,
		InitialDelay:          defaultInitialDelay,
		TerminationGraceScans: defaultTerminationGraceScans,
		MaxTrackedProcesses:   defaultMaxTrackedProcesses,
		ProcFSPath:            defaultProcFSPath,
		Runtimes:              slices.Clone(supportedRuntimes),
		MaxCmdlineBytes:       defaultMaxCmdlineBytes,
		RuntimeDetection: RuntimeDetectionConfig{
			BinaryCacheTTL:  defaultBinaryCacheTTL,
			BinaryCacheSize: defaultBinaryCacheSize,
		},
		Lifecycle: LifecycleConfig{
			Enabled:    true,
			BufferSize: defaultLifecycleBufferSize,
		},
		ClockTicks:  defaultClockTicks,
		CommandLine: CommandLineConfig{Enabled: true, MaxArguments: defaultMaxCommandArgs},
		Network:     NetworkConfig{Enabled: true, MaxPortsPerProcess: 128, MaxSeries: 2000},
	}
}

func (cfg *Config) Validate() error {
	if cfg.CollectionInterval < 2*time.Second {
		return fmt.Errorf("collection_interval must be at least 2s")
	}
	if cfg.ReportInterval < cfg.CollectionInterval {
		return fmt.Errorf("report_interval must be at least collection_interval")
	}
	if cfg.InitialDelay < 0 {
		return fmt.Errorf("initial_delay must not be negative")
	}
	if cfg.TerminationGraceScans < 1 {
		return fmt.Errorf("termination_grace_scans must be at least 1")
	}
	if cfg.MaxTrackedProcesses < 1 {
		return fmt.Errorf("max_tracked_processes must be at least 1")
	}
	if cfg.MaxCmdlineBytes < 1 || cfg.MaxCmdlineBytes > maxCmdlineBytes {
		return fmt.Errorf("max_cmdline_bytes must be between 1 and %d", maxCmdlineBytes)
	}
	if !filepath.IsAbs(cfg.ProcFSPath) {
		return fmt.Errorf("procfs_path must be absolute")
	}
	if cfg.HostMachineIDPath != "" && !filepath.IsAbs(cfg.HostMachineIDPath) {
		return fmt.Errorf("host_machine_id_path must be absolute")
	}
	if len(cfg.Runtimes) == 0 {
		return fmt.Errorf("at least one runtime must be enabled")
	}
	seen := make(map[string]struct{}, len(cfg.Runtimes))
	for _, runtimeName := range cfg.Runtimes {
		if !slices.Contains(supportedRuntimes, runtimeName) {
			return fmt.Errorf("unsupported runtime %q", runtimeName)
		}
		if _, ok := seen[runtimeName]; ok {
			return fmt.Errorf("runtime %q is configured more than once", runtimeName)
		}
		seen[runtimeName] = struct{}{}
	}
	if cfg.RuntimeDetection.BinaryCacheTTL <= 0 {
		return fmt.Errorf("runtime_detection.binary_cache_ttl must be positive")
	}
	if cfg.RuntimeDetection.BinaryCacheSize < 1 {
		return fmt.Errorf("runtime_detection.binary_cache_size must be at least 1")
	}
	if cfg.Lifecycle.BufferSize < 1 {
		return fmt.Errorf("lifecycle.buffer_size must be at least 1")
	}
	if cfg.ClockTicks == 0 {
		return fmt.Errorf("clock_ticks must be positive")
	}
	if cfg.CommandLine.MaxArguments < 1 {
		return fmt.Errorf("command_line.max_arguments must be at least 1")
	}
	if cfg.Network.MaxPortsPerProcess < 1 {
		return fmt.Errorf("network.max_ports_per_process must be at least 1")
	}
	if cfg.Network.MaxSeries < 1 {
		return fmt.Errorf("network.max_series must be at least 1")
	}
	if err := cfg.Kubernetes.Validate(); err != nil {
		return err
	}
	return nil
}
