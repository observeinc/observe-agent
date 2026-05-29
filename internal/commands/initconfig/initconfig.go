/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package initconfig

import (
	"fmt"
	"os"

	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	config_path string

	// Top-level
	token                 string
	observe_url           string
	cloud_resource_detectors []string
	attributes            map[string]string
	resource_attributes   map[string]string
	omit_base_components  bool
	agent_local_file_path string

	// application.RED_metrics
	application_RED_metrics_enabled                                       bool
	application_RED_metrics_only_generate_for_service_entrypoint_spans    bool
	application_RED_metrics_resource_dimensions                           []string
	application_RED_metrics_span_dimensions                               []string

	// health_check
	health_check_enabled  bool
	health_check_endpoint string
	health_check_path     string

	// forwarding
	forwarding_enabled                            bool
	forwarding_endpoints_http                     string
	forwarding_endpoints_grpc                     string
	forwarding_metrics_format                     string
	forwarding_metrics_convert_cumulative_to_delta bool
	forwarding_traces_max_span_duration           string

	// internal_telemetry
	internal_telemetry_enabled         bool
	internal_telemetry_metrics_enabled bool
	internal_telemetry_metrics_host    string
	internal_telemetry_metrics_port    int
	internal_telemetry_metrics_level   string
	internal_telemetry_logs_enabled    bool
	internal_telemetry_logs_level      string
	internal_telemetry_logs_encoding   string

	// self_monitoring
	self_monitoring_enabled               bool
	self_monitoring_fleet_enabled         bool
	self_monitoring_fleet_interval        string
	self_monitoring_fleet_config_interval string

	// host_monitoring
	host_monitoring_enabled                       bool
	host_monitoring_logs_enabled                  bool
	host_monitoring_logs_include                  []string
	host_monitoring_logs_exclude                  []string
	host_monitoring_logs_auto_multiline_detection bool
	host_monitoring_metrics_host_enabled          bool
	host_monitoring_metrics_process_enabled       bool

	// exporters
	exporters_sending_queue_batch_enabled        bool
	exporters_sending_queue_batch_max_size       int
	exporters_emit_prometheus_target_info_metric bool
)

func NewConfigureCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "init-config",
		Short: "Initialize agent configuration",
		Long:  `This command takes in parameters and creates an initialized observe agent configuration file. Will overwrite existing config file and should only be used to initialize.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var f *os.File
			if v.GetBool("print") {
				f = os.Stdout
			} else {
				var outputPath string
				if config_path != "" {
					outputPath = config_path
				} else {
					outputPath = v.ConfigFileUsed()
				}
				var err error
				f, err = os.Create(outputPath)
				if err != nil {
					return err
				}
				defer f.Close()
				fmt.Printf("Writing configuration values to %s...\n\n", outputPath)
			}
			agentConfig, err := config.AgentConfigFromViper(v)
			if err != nil {
				return err
			}
			writeConfigFile(f, agentConfig, v.GetBool("include-defaults"))
			return nil
		},
	}
}

func init() {
	v := viper.GetViper()
	configureCmd := NewConfigureCmd(v)
	RegisterConfigFlags(configureCmd, v)
	root.RootCmd.AddCommand(configureCmd)
}

// bindFlag is a small helper to keep flag declarations terse: registering a
// pflag and binding it to viper is always the same two-line ritual. Callers
// that need a non-zero default that won't already be set via SetViperDefaults
// (e.g. fields whose struct lacks a `default:` tag) should additionally call
// v.SetDefault — see the self_monitoring/host_monitoring blocks below.
func bindFlag(cmd *cobra.Command, v *viper.Viper, name string) {
	v.BindPFlag(name, cmd.PersistentFlags().Lookup(name))
}

func RegisterConfigFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().StringVarP(&config_path, "config_path", "", "", "Path to write config output file to")
	cmd.Flags().Bool("print", false, "Print the configuration to stdout instead of writing to a file")
	v.BindPFlag("print", cmd.Flags().Lookup("print"))
	cmd.Flags().Bool("include-defaults", false, "Include the names and default values for unset config options.")
	v.BindPFlag("include-defaults", cmd.Flags().Lookup("include-defaults"))

	// ---- Top-level ----

	cmd.PersistentFlags().StringVar(&token, "token", "", "Observe token")
	bindFlag(cmd, v, "token")

	cmd.PersistentFlags().StringVar(&observe_url, "observe_url", "", "Observe data collection url")
	bindFlag(cmd, v, "observe_url")

	cmd.PersistentFlags().StringSliceVar(&cloud_resource_detectors, "cloud_resource_detectors", []string{}, "The cloud environments from which to detect resources")
	bindFlag(cmd, v, "cloud_resource_detectors")

	cmd.PersistentFlags().StringToStringVar(&attributes, "attributes", map[string]string{}, "Global telemetry attributes (key=value, comma-separated). Distinct from resource_attributes — these are applied at the telemetry-record level.")
	bindFlag(cmd, v, "attributes")

	cmd.PersistentFlags().StringToStringVar(&resource_attributes, "resource_attributes", map[string]string{}, "Global resource attributes (key=value, comma-separated). Applied at the OTel resource level.")
	bindFlag(cmd, v, "resource_attributes")

	cmd.PersistentFlags().BoolVar(&omit_base_components, "omit_base_components", false, "Skip emitting the bundled base OTel pipeline components (advanced; only set if you know you need it)")
	bindFlag(cmd, v, "omit_base_components")

	cmd.PersistentFlags().StringVar(&agent_local_file_path, "agent_local_file_path", "", "Path the agent uses for local-file persistence (advanced)")
	bindFlag(cmd, v, "agent_local_file_path")

	// ---- application.RED_metrics ----

	cmd.PersistentFlags().BoolVar(&application_RED_metrics_enabled, "application::RED_metrics::enabled", false, "Enable RED metrics generation for application traces")
	bindFlag(cmd, v, "application::RED_metrics::enabled")

	cmd.PersistentFlags().BoolVar(&application_RED_metrics_only_generate_for_service_entrypoint_spans, "application::RED_metrics::only_generate_for_service_entrypoint_spans", false, "When generating RED metrics, skip spans that are not service entrypoint spans (kind Server / Consumer or DB/messaging client calls)")
	bindFlag(cmd, v, "application::RED_metrics::only_generate_for_service_entrypoint_spans")

	cmd.PersistentFlags().StringSliceVar(&application_RED_metrics_resource_dimensions, "application::RED_metrics::resource_dimensions", nil, "Resource attributes to include as RED metric dimensions (defaults to service.namespace, service.version, deployment.environment)")
	bindFlag(cmd, v, "application::RED_metrics::resource_dimensions")

	cmd.PersistentFlags().StringSliceVar(&application_RED_metrics_span_dimensions, "application::RED_metrics::span_dimensions", nil, "Span attributes to include as RED metric dimensions (defaults to peer.db.name, peer.messaging.system, otel.status_description, observe.status_code)")
	bindFlag(cmd, v, "application::RED_metrics::span_dimensions")

	// ---- health_check ----

	cmd.PersistentFlags().BoolVar(&health_check_enabled, "health_check::enabled", true, "Enable the agent health-check HTTP endpoint")
	bindFlag(cmd, v, "health_check::enabled")

	cmd.PersistentFlags().StringVar(&health_check_endpoint, "health_check::endpoint", "localhost:13133", "Address the health-check endpoint binds to")
	bindFlag(cmd, v, "health_check::endpoint")

	cmd.PersistentFlags().StringVar(&health_check_path, "health_check::path", "/status", "HTTP path the health-check endpoint serves")
	bindFlag(cmd, v, "health_check::path")

	// ---- forwarding ----

	cmd.PersistentFlags().BoolVar(&forwarding_enabled, "forwarding::enabled", true, "Enable the OTLP receivers for forwarding application telemetry through the agent")
	bindFlag(cmd, v, "forwarding::enabled")

	cmd.PersistentFlags().StringVar(&forwarding_endpoints_http, "forwarding::endpoints::http", "localhost:4318", "Address the OTLP HTTP receiver binds to")
	bindFlag(cmd, v, "forwarding::endpoints::http")

	cmd.PersistentFlags().StringVar(&forwarding_endpoints_grpc, "forwarding::endpoints::grpc", "localhost:4317", "Address the OTLP gRPC receiver binds to")
	bindFlag(cmd, v, "forwarding::endpoints::grpc")

	cmd.PersistentFlags().StringVar(&forwarding_metrics_format, "forwarding::metrics::output_format", "", "Format for sending app metrics to Observe, valid options are 'prometheus' and 'otel'")
	bindFlag(cmd, v, "forwarding::metrics::output_format")

	cmd.PersistentFlags().BoolVar(&forwarding_metrics_convert_cumulative_to_delta, "forwarding::metrics::convert_cumulative_to_delta", false, "Convert cumulative metrics to delta before forwarding (only valid when output_format=otel)")
	bindFlag(cmd, v, "forwarding::metrics::convert_cumulative_to_delta")

	cmd.PersistentFlags().StringVar(&forwarding_traces_max_span_duration, "forwarding::traces::max_span_duration", "1h", "Drop spans whose duration exceeds this value (e.g. '1h', '15m', or 'none' to disable)")
	bindFlag(cmd, v, "forwarding::traces::max_span_duration")

	// ---- internal_telemetry ----

	cmd.PersistentFlags().BoolVar(&internal_telemetry_enabled, "internal_telemetry::enabled", true, "Enable internal telemetry (the agent observing itself)")
	bindFlag(cmd, v, "internal_telemetry::enabled")

	cmd.PersistentFlags().BoolVar(&internal_telemetry_metrics_enabled, "internal_telemetry::metrics::enabled", true, "Enable internal Prometheus metrics about the agent's own pipelines")
	bindFlag(cmd, v, "internal_telemetry::metrics::enabled")

	cmd.PersistentFlags().StringVar(&internal_telemetry_metrics_host, "internal_telemetry::metrics::host", "localhost", "Host the internal-telemetry metrics endpoint binds to")
	bindFlag(cmd, v, "internal_telemetry::metrics::host")

	cmd.PersistentFlags().IntVar(&internal_telemetry_metrics_port, "internal_telemetry::metrics::port", 8888, "Port the internal-telemetry metrics endpoint binds to")
	bindFlag(cmd, v, "internal_telemetry::metrics::port")

	cmd.PersistentFlags().StringVar(&internal_telemetry_metrics_level, "internal_telemetry::metrics::level", "detailed", "Internal telemetry verbosity level (basic / normal / detailed)")
	bindFlag(cmd, v, "internal_telemetry::metrics::level")

	cmd.PersistentFlags().BoolVar(&internal_telemetry_logs_enabled, "internal_telemetry::logs::enabled", true, "Enable the agent's own log output")
	bindFlag(cmd, v, "internal_telemetry::logs::enabled")

	cmd.PersistentFlags().StringVar(&internal_telemetry_logs_level, "internal_telemetry::logs::level", "${env:OTEL_LOG_LEVEL}", "Agent log level (debug, info, warn, error). Defaults to the OTEL_LOG_LEVEL env var.")
	bindFlag(cmd, v, "internal_telemetry::logs::level")

	cmd.PersistentFlags().StringVar(&internal_telemetry_logs_encoding, "internal_telemetry::logs::encoding", "console", "Agent log encoding ('console' or 'json')")
	bindFlag(cmd, v, "internal_telemetry::logs::encoding")

	// ---- self_monitoring ----

	cmd.PersistentFlags().BoolVar(&self_monitoring_enabled, "self_monitoring::enabled", true, "Enable self monitoring")
	bindFlag(cmd, v, "self_monitoring::enabled")
	v.SetDefault("self_monitoring::enabled", true)

	cmd.PersistentFlags().BoolVar(&self_monitoring_fleet_enabled, "self_monitoring::fleet::enabled", true, "Enable fleet heartbeat")
	bindFlag(cmd, v, "self_monitoring::fleet::enabled")
	// Without this SetDefault, viper.Unmarshal returns the SetViperDefaults value
	// (zero — false) instead of the cobra default, silently disabling fleet
	// heartbeats whenever the flag isn't passed explicitly. Matches the pattern
	// used for self_monitoring::enabled and the host_monitoring flags.
	v.SetDefault("self_monitoring::fleet::enabled", true)

	cmd.PersistentFlags().StringVar(&self_monitoring_fleet_interval, "self_monitoring::fleet::interval", "", "Fleet heartbeat interval (e.g., '5m', '1h')")
	bindFlag(cmd, v, "self_monitoring::fleet::interval")

	cmd.PersistentFlags().StringVar(&self_monitoring_fleet_config_interval, "self_monitoring::fleet::config_interval", "", "Fleet config heartbeat interval (e.g., '30m', '24h')")
	bindFlag(cmd, v, "self_monitoring::fleet::config_interval")

	// ---- host_monitoring ----

	cmd.PersistentFlags().BoolVar(&host_monitoring_enabled, "host_monitoring::enabled", true, "Enable host monitoring")
	bindFlag(cmd, v, "host_monitoring::enabled")
	v.SetDefault("host_monitoring::enabled", true)

	cmd.PersistentFlags().BoolVar(&host_monitoring_logs_enabled, "host_monitoring::logs::enabled", true, "Enable host monitoring logs")
	bindFlag(cmd, v, "host_monitoring::logs::enabled")
	v.SetDefault("host_monitoring::logs::enabled", true)

	cmd.PersistentFlags().StringSliceVar(&host_monitoring_logs_include, "host_monitoring::logs::include", nil, "Set host monitoring log include paths")
	bindFlag(cmd, v, "host_monitoring::logs::include")

	cmd.PersistentFlags().StringSliceVar(&host_monitoring_logs_exclude, "host_monitoring::logs::exclude", nil, "Host monitoring log paths to exclude (applied after include)")
	bindFlag(cmd, v, "host_monitoring::logs::exclude")

	cmd.PersistentFlags().BoolVar(&host_monitoring_logs_auto_multiline_detection, "host_monitoring::logs::auto_multiline_detection", false, "Enable host monitoring log auto multiline detection")
	bindFlag(cmd, v, "host_monitoring::logs::auto_multiline_detection")
	v.SetDefault("host_monitoring::logs::auto_multiline_detection", false)

	cmd.PersistentFlags().BoolVar(&host_monitoring_metrics_host_enabled, "host_monitoring::metrics::host::enabled", true, "Enable host monitoring host metrics")
	bindFlag(cmd, v, "host_monitoring::metrics::host::enabled")
	v.SetDefault("host_monitoring::metrics::host::enabled", true)

	cmd.PersistentFlags().BoolVar(&host_monitoring_metrics_process_enabled, "host_monitoring::metrics::process::enabled", false, "Enable host monitoring process metrics")
	bindFlag(cmd, v, "host_monitoring::metrics::process::enabled")
	v.SetDefault("host_monitoring::metrics::process::enabled", false)

	// ---- exporters ----

	cmd.PersistentFlags().BoolVar(&exporters_sending_queue_batch_enabled, "exporters::sending_queue_batch::enabled", false, "Enable batching at the OTel exporter sending queue (advanced)")
	bindFlag(cmd, v, "exporters::sending_queue_batch::enabled")

	cmd.PersistentFlags().IntVar(&exporters_sending_queue_batch_max_size, "exporters::sending_queue_batch::max_size", 41943040, "Max batch size in bytes for the exporter sending queue")
	bindFlag(cmd, v, "exporters::sending_queue_batch::max_size")

	cmd.PersistentFlags().BoolVar(&exporters_emit_prometheus_target_info_metric, "exporters::emit_prometheus_target_info_metric", false, "Emit the synthetic 'target_info' metric when exporting Prometheus")
	bindFlag(cmd, v, "exporters::emit_prometheus_target_info_metric")
}
