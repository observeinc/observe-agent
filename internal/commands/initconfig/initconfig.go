/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package initconfig

import (
	"embed"
	"fmt"
	"html/template"
	"os"

	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	config_path                             string
	token                                   string
	observe_url                             string
	cloud_resource_detectors                []string
	self_monitoring_enabled                 bool
	host_monitoring_enabled                 bool
	host_monitoring_logs_enabled            bool
	host_monitoring_logs_include            []string
	host_monitoring_metrics_host_enabled    bool
	host_monitoring_metrics_process_enabled bool
	//go:embed observe-agent.tmpl
	configTemplateFS embed.FS
)

const configTemplate = "observe-agent.tmpl"

type FlatAgentConfig struct {
	Token                                 string
	ObserveURL                            string
	CloudResourceDetectors                []string
	SelfMonitoring_Enabled                bool
	HostMonitoring_Enabled                bool
	HostMonitoring_LogsEnabled            bool
	HostMonitoring_LogsInclude            []string
	HostMonitoring_Metrics_HostEnabled    bool
	HostMonitoring_Metrics_ProcessEnabled bool
}

func NewConfigureCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "init-config",
		Short: "Initialize agent configuration",
		Long:  `This command takes in parameters and creates an initialized observe agent configuration file. Will overwrite existing config file and should only be used to initialize.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configValues := FlatAgentConfig{
				Token:                                 v.GetString("token"),
				ObserveURL:                            v.GetString("observe_url"),
				CloudResourceDetectors:                v.GetStringSlice("cloud_resource_detectors"),
				SelfMonitoring_Enabled:                v.GetBool("self_monitoring::enabled"),
				HostMonitoring_Enabled:                v.GetBool("host_monitoring::enabled"),
				HostMonitoring_LogsEnabled:            v.GetBool("host_monitoring::logs::enabled"),
				HostMonitoring_Metrics_HostEnabled:    v.GetBool("host_monitoring::metrics::host::enabled"),
				HostMonitoring_Metrics_ProcessEnabled: v.GetBool("host_monitoring::metrics::process::enabled"),
			}
			if configValues.HostMonitoring_LogsEnabled {
				configValues.HostMonitoring_LogsInclude = v.GetStringSlice("host_monitoring::logs::include")
			}
			var outputPath string
			if config_path != "" {
				outputPath = config_path
			} else {
				outputPath = v.ConfigFileUsed()
			}
			f, err := os.Create(outputPath)
			if err != nil {
				return err
			}
			defer f.Close()
			t := template.Must(template.New(configTemplate).ParseFS(configTemplateFS, configTemplate))
			if err := t.ExecuteTemplate(f, configTemplate, configValues); err != nil {
				return err
			}
			fmt.Printf("Writing configuration values to %s...\n\n", outputPath)
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

func RegisterConfigFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().StringVarP(&config_path, "config_path", "", "", "Path to write config output file to")
	cmd.PersistentFlags().StringVar(&token, "token", "", "Observe token")
	cmd.PersistentFlags().StringVar(&observe_url, "observe_url", "", "Observe data collection url")
	cmd.PersistentFlags().StringSliceVar(&cloud_resource_detectors, "cloud_resource_detectors", []string{}, "Cloud resource detectors")
	cmd.PersistentFlags().BoolVar(&self_monitoring_enabled, "self_monitoring::enabled", true, "Enable self monitoring")
	cmd.PersistentFlags().BoolVar(&host_monitoring_enabled, "host_monitoring::enabled", true, "Enable host monitoring")
	cmd.PersistentFlags().BoolVar(&host_monitoring_logs_enabled, "host_monitoring::logs::enabled", true, "Enable host monitoring logs")
	cmd.PersistentFlags().StringSliceVar(&host_monitoring_logs_include, "host_monitoring::logs::include", nil, "Set host monitoring log include paths")
	cmd.PersistentFlags().BoolVar(&host_monitoring_metrics_host_enabled, "host_monitoring::metrics::host::enabled", true, "Enable host monitoring host metrics")
	cmd.PersistentFlags().BoolVar(&host_monitoring_metrics_process_enabled, "host_monitoring::metrics::process::enabled", false, "Enable host monitoring process metrics")
	v.BindPFlag("token", cmd.PersistentFlags().Lookup("token"))
	v.BindPFlag("observe_url", cmd.PersistentFlags().Lookup("observe_url"))
	v.BindPFlag("cloud_resource_detectors", cmd.PersistentFlags().Lookup("cloud_resource_detectors"))
	v.BindPFlag("self_monitoring::enabled", cmd.PersistentFlags().Lookup("self_monitoring::enabled"))
	v.BindPFlag("host_monitoring::enabled", cmd.PersistentFlags().Lookup("host_monitoring::enabled"))
	v.BindPFlag("host_monitoring::logs::enabled", cmd.PersistentFlags().Lookup("host_monitoring::logs::enabled"))
	v.BindPFlag("host_monitoring::logs::include", cmd.PersistentFlags().Lookup("host_monitoring::logs::include"))
	v.BindPFlag("host_monitoring::metrics::host::enabled", cmd.PersistentFlags().Lookup("host_monitoring::metrics::host::enabled"))
	v.BindPFlag("host_monitoring::metrics::process::enabled", cmd.PersistentFlags().Lookup("host_monitoring::metrics::process::enabled"))
}
