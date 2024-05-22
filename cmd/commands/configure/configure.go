/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package configure

import (
	"fmt"
	"observe/agent/cmd"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Configure agent",
	Long:  `This command takes in parameters and writes them to the observe agent's configuration file.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Running configuration...\n\n")
	},
}

func init() {
	cmd.RootCmd.AddCommand(configureCmd)
}
