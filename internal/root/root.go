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
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
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

	RootCmd.PersistentFlags().StringVar(&CfgFile, "config", "", "config file path")
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
