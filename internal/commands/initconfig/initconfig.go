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
	config_path                             string
	token                                   string
	observe_url                             string
	cloud_resource_detectors                []string
	resource_attributes                     map[string]string
	self_monitoring_enabled                 bool
	host_monitoring_enabled                 bool
	host_monitoring_logs_enabled            bool
	host_monitoring_logs_include            []string
	host_monitoring_metrics_host_enabled    bool
	host_monitoring_metrics_process_enabled bool
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

func RegisterConfigFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().StringVarP(&config_path, "config_path", "", "", "Path to write config output file to")
	cmd.Flags().Bool("print", false, "Print the configuration to stdout instead of writing to a file")
	v.BindPFlag("print", cmd.Flags().Lookup("print"))
	cmd.Flags().Bool("include-defaults", false, "Include the names and default values for unset config options.")
	v.BindPFlag("include-defaults", cmd.Flags().Lookup("include-defaults"))

	cmd.PersistentFlags().StringVar(&token, "token", "", "Observe token")
	v.BindPFlag("token", cmd.PersistentFlags().Lookup("token"))

	cmd.PersistentFlags().StringVar(&observe_url, "observe_url", "", "Observe data collection url")
	v.BindPFlag("observe_url", cmd.PersistentFlags().Lookup("observe_url"))

	cmd.PersistentFlags().StringSliceVar(&cloud_resource_detectors, "cloud_resource_detectors", []string{}, "The cloud environments from which to detect resources")
	v.BindPFlag("cloud_resource_detectors", cmd.PersistentFlags().Lookup("cloud_resource_detectors"))

	cmd.PersistentFlags().StringToStringVar(&resource_attributes, "resource_attributes", map[string]string{}, "The cloud environments from which to detect resources")
	v.BindPFlag("resource_attributes", cmd.PersistentFlags().Lookup("resource_attributes"))

	cmd.PersistentFlags().BoolVar(&self_monitoring_enabled, "self_monitoring::enabled", true, "Enable self monitoring")
	v.BindPFlag("self_monitoring::enabled", cmd.PersistentFlags().Lookup("self_monitoring::enabled"))
	v.SetDefault("self_monitoring::enabled", true)

	cmd.PersistentFlags().BoolVar(&host_monitoring_enabled, "host_monitoring::enabled", true, "Enable host monitoring")
	v.BindPFlag("host_monitoring::enabled", cmd.PersistentFlags().Lookup("host_monitoring::enabled"))
	v.SetDefault("host_monitoring::enabled", true)

	cmd.PersistentFlags().BoolVar(&host_monitoring_logs_enabled, "host_monitoring::logs::enabled", true, "Enable host monitoring logs")
	v.BindPFlag("host_monitoring::logs::enabled", cmd.PersistentFlags().Lookup("host_monitoring::logs::enabled"))
	v.SetDefault("host_monitoring::logs::enabled", true)

	cmd.PersistentFlags().StringSliceVar(&host_monitoring_logs_include, "host_monitoring::logs::include", nil, "Set host monitoring log include paths")
	v.BindPFlag("host_monitoring::logs::include", cmd.PersistentFlags().Lookup("host_monitoring::logs::include"))

	cmd.PersistentFlags().BoolVar(&host_monitoring_metrics_host_enabled, "host_monitoring::metrics::host::enabled", true, "Enable host monitoring host metrics")
	v.BindPFlag("host_monitoring::metrics::host::enabled", cmd.PersistentFlags().Lookup("host_monitoring::metrics::host::enabled"))
	v.SetDefault("host_monitoring::metrics::host::enabled", true)

	cmd.PersistentFlags().BoolVar(&host_monitoring_metrics_process_enabled, "host_monitoring::metrics::process::enabled", false, "Enable host monitoring process metrics")
	v.BindPFlag("host_monitoring::metrics::process::enabled", cmd.PersistentFlags().Lookup("host_monitoring::metrics::process::enabled"))
	v.SetDefault("host_monitoring::metrics::process::enabled", false)
}
