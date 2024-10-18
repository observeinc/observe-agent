/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package root

import (
	"fmt"
	"os"

	"github.com/observeinc/observe-agent/internal/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var CfgFile string

// RootCmd represents the base command when called without any subcommands
var RootCmd = &cobra.Command{
	Use:   "observe-agent",
	Short: "Observe distribution of OTEL Collector",
	Long: `Observe distribution of OTEL Collector along with CLI utils to help with setup
and maintenance. To start the agent, run: observe-agent start`,
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

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	RootCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file path")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	RootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

// InitConfig reads in config file and ENV variables if set.
func InitConfig() {
	// Some keys in OTEL component configs use "." as part of the key but viper ends up parsing that into
	// a subobject since the default key delimiter is "." which causes config validation to fail.
	// We set it to "::" here to prevent that behavior. This call modifies the global viper instance.
	viper.SetOptions(viper.KeyDelimiter("::"))
	if CfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(CfgFile)
	} else {
		viper.AddConfigPath(config.GetDefaultAgentPath())
		viper.SetConfigType("yaml")
		viper.SetConfigName("observe-agent")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}
