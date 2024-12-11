/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"context"
	"fmt"
	"os"

	"github.com/observeinc/observe-agent/internal/commands/start"
	logger "github.com/observeinc/observe-agent/internal/commands/util"
	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Prints the full configuration for this agent.",
	Long: `This command prints all configuration for this agent including any additional
OTEL configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := logger.WithCtx(context.Background(), logger.GetNop())
		configFilePaths, cleanup, err := start.SetupAndGetConfigFiles(ctx)
		if err != nil {
			return err
		}
		if cleanup != nil {
			defer cleanup()
		}
		agentConfig, err := config.AgentConfigFromViper(viper.GetViper())
		if err != nil {
			return err
		}
		agentConfigYaml, err := yaml.Marshal(agentConfig)
		if err != nil {
			return err
		}
		fmt.Printf("# ======== computed agent config\n")
		fmt.Println(string(agentConfigYaml) + "\n")
		agentConfigFile := viper.ConfigFileUsed()
		if agentConfigFile != "" {
			configFilePaths = append([]string{agentConfigFile}, configFilePaths...)
		}
		for _, filePath := range configFilePaths {
			file, err := os.ReadFile(filePath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error reading config file %s: %s", filePath, err.Error())
			} else {
				fmt.Printf("# ======== config file %s\n", filePath)
				fmt.Println(string(file))
			}
		}
		return nil
	},
}

func init() {
	root.RootCmd.AddCommand(configCmd)
}
