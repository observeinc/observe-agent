/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package initconfig

import (
	"embed"
	"fmt"
	"html/template"
	"observe-agent/cmd"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	config_path                     string
	token                           string
	observe_url                     string
	self_monitoring_enabled         bool
	host_monitoring_enabled         bool
	host_monitoring_logs_enabled    bool
	host_monitoring_metrics_enabled bool
	//go:embed observe-agent.tmpl
	configTemplateFS embed.FS
)

const configTemplate = "observe-agent.tmpl"

type FlatAgentConfig struct {
	Token                         string
	ObserveURL                    string
	SelfMonitoring_Enabled        bool
	HostMonitoring_Enabled        bool
	HostMonitoring_LogsEnabled    bool
	HostMonitoring_MetricsEnabled bool
}

func NewConfigureCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init-config",
		Short: "Initialize agent configuration",
		Long:  `This command takes in parameters and creates an initialized observe agent configuration file. Will overwrite existing config file and should only be used to initialize.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			configValues := FlatAgentConfig{
				Token:                         viper.GetString("token"),
				ObserveURL:                    viper.GetString("observe_url"),
				SelfMonitoring_Enabled:        viper.GetBool("self_monitoring::enabled"),
				HostMonitoring_Enabled:        viper.GetBool("host_monitoring::enabled"),
				HostMonitoring_LogsEnabled:    viper.GetBool("host_monitoring::logs::enabled"),
				HostMonitoring_MetricsEnabled: viper.GetBool("host_monitoring::metrics::enabled"),
			}
			var outputPath string
			if config_path != "" {
				outputPath = config_path
			} else {
				outputPath = viper.ConfigFileUsed()
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
	configureCmd := NewConfigureCmd()
	RegisterConfigFlags(configureCmd)
	cmd.RootCmd.AddCommand(configureCmd)
}

func RegisterConfigFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&config_path, "config_path", "", "", "Path to write config output file to")
	cmd.PersistentFlags().StringVar(&token, "token", "", "Observe token")
	cmd.PersistentFlags().StringVar(&observe_url, "observe_url", "", "Observe data collection url")
	cmd.PersistentFlags().BoolVar(&self_monitoring_enabled, "self_monitoring::enabled", true, "Enable self monitoring")
	cmd.PersistentFlags().BoolVar(&host_monitoring_enabled, "host_monitoring::enabled", true, "Enable host monitoring")
	cmd.PersistentFlags().BoolVar(&host_monitoring_logs_enabled, "host_monitoring::logs::enabled", true, "Enable host monitoring logs")
	cmd.PersistentFlags().BoolVar(&host_monitoring_metrics_enabled, "host_monitoring::metrics::enabled", true, "Enable host monitoring metrics")
	viper.BindPFlag("token", cmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("observe_url", cmd.PersistentFlags().Lookup("observe_url"))
	viper.BindPFlag("self_monitoring::enabled", cmd.PersistentFlags().Lookup("self_monitoring::enabled"))
	viper.BindPFlag("host_monitoring::enabled", cmd.PersistentFlags().Lookup("host_monitoring::enabled"))
	viper.BindPFlag("host_monitoring::logs::enabled", cmd.PersistentFlags().Lookup("host_monitoring::logs::enabled"))
	viper.BindPFlag("host_monitoring::metrics::enabled", cmd.PersistentFlags().Lookup("host_monitoring::metrics::enabled"))
}
