/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package root

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/observeinc/observe-agent/build"
	"github.com/observeinc/observe-agent/internal/commands/util"
	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/connections"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var CfgFile string
var configMode string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "observe-agent",
	Short: "Observe distribution of OTEL Collector",
	Long: `Observe distribution of OTEL Collector along with CLI utils to help with setup
and maintenance. To start the agent, run: observe-agent start`,
	Version: build.Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := RootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(InitConfig)

	flags := RootCmd.PersistentFlags()
	flags.StringVar(&CfgFile, "observe-config", "", "observe-agent config file path")
	flags.StringVar(&configMode, "config-mode", "", "The mode to use for bundled config. Valid values are Linux, Docker, Mac, and Windows.")
	flags.MarkHidden("config-mode")
	observecol.AddConfigFlags(flags)
	observecol.AddFeatureGateFlag(flags)
}

func setConfigMode() {
	var overrides bundledconfig.ConfigTemplates
	switch strings.ToLower(configMode) {
	case "":
		return
	case "linux":
		overrides = bundledconfig.LinuxTemplateFS
	case "docker":
		overrides = bundledconfig.DockerTemplateFS
	case "mac":
		overrides = bundledconfig.MacOSTemplateFS
	case "windows":
		overrides = bundledconfig.WindowsTemplateFS
	default:
		fmt.Fprintf(os.Stderr, "Invalid config mode specified: %s. Valid values are Linux, Docker, Mac, and Windows.\n", configMode)
		os.Exit(1)
	}
	// Set the template overrides for all connections
	for _, conn := range connections.AllConnectionTypes {
		conn.ApplyOptions(connections.WithConfigTemplateOverrides(overrides))
	}

}

// InitConfig reads in config file and ENV variables if set.
func InitConfig() {
	setConfigMode()
	ctx := logger.WithCtx(context.Background(), logger.Get())
	// Some keys in OTEL component configs use "." as part of the key but viper ends up parsing that into
	// a subobject since the default key delimiter is "." which causes config validation to fail.
	// We set it to "::" here to prevent that behavior. This call modifies the global viper instance.
	viper.SetOptions(viper.KeyDelimiter("::"))
	if CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(CfgFile)
	} else {
		viper.AddConfigPath(connections.GetDefaultAgentPath())
		viper.SetConfigType("yaml")
		viper.SetConfigName("observe-agent")
	}

	// TODO consider setting this in our next major release to scope all agent env vars:
	// viper.SetEnvPrefix("OBSERVE")
	viper.AutomaticEnv() // read in environment variables that match

	config.SetViperDefaults(viper.GetViper(), "::")

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore this error.
		} else {
			fmt.Fprintln(os.Stderr, "error reading config file:", err)
		}
	}

	// Apply feature gates
	observecol.ApplyFeatureGates(ctx)

	// Set up env vars
	if err := setEnvVars(); err != nil {
		fmt.Fprintln(os.Stderr, "error setting env vars:", err)
	}
}

func setEnvVars() error {
	collector_url, token, debug := viper.GetString("observe_url"), viper.GetString("token"), viper.GetBool("debug")
	// Ensure the collector url does not end with a slash for consistency. This will allow endpoints to be configured like:
	// "${env:OBSERVE_COLLECTOR_URL}/v1/kubernetes/v1/entity"
	// without worrying about a double slash.
	collector_url = strings.TrimRight(collector_url, "/")
	otelEndpoint := util.JoinUrl(collector_url, "/v2/otel")
	promEndpoint := util.JoinUrl(collector_url, "/v1/prometheus")
	// Setting values from the Observe agent config as env vars to fill in the OTEL collector config
	os.Setenv("OBSERVE_COLLECTOR_URL", collector_url)
	os.Setenv("OBSERVE_OTEL_ENDPOINT", otelEndpoint)
	os.Setenv("OBSERVE_PROMETHEUS_ENDPOINT", promEndpoint)
	os.Setenv("OBSERVE_AUTHORIZATION_HEADER", "Bearer "+token)
	os.Setenv("FILESTORAGE_PATH", getDefaultFilestoragePath())

	configFile := viper.ConfigFileUsed()
	if configFile != "" {
		os.Setenv("OBSERVE_AGENT_CONFIG_PATH", configFile)
	}

	// Default TRACE_TOKEN to be the value of the configured token if it's not set. This allows for users to upgrade to
	// direct write tracing with ingest tokens in kubernetes without breaking backwards compatibility in our helm chart.
	// TODO: remove this once our helm chart no longer supports TRACE_TOKEN
	if os.Getenv("TRACE_TOKEN") == "" {
		os.Setenv("TRACE_TOKEN", token)
	}

	if os.Getenv("OTEL_LOG_LEVEL") == "" {
		if debug {
			os.Setenv("OTEL_LOG_LEVEL", "DEBUG")
		} else {
			os.Setenv("OTEL_LOG_LEVEL", "INFO")
		}
	}
	return nil
}

func getDefaultFilestoragePath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return "/var/lib/observe-agent/filestorage"
	case "windows":
		return os.ExpandEnv("$ProgramData\\Observe\\observe-agent\\filestorage")
	case "linux":
		return "/var/lib/observe-agent/filestorage"
	default:
		return "/var/lib/observe-agent/filestorage"
	}
}
