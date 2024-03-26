/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	observeotel "observe/agent/cmd/connections/otelcollector"
	"sync"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the Observe agent process.",
	Long: `The Observe agent is based on the OpenTelemetry Collector. 
This command reads in the local config and env vars and starts the 
collector on the current host.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var wg sync.WaitGroup
		wg.Wait()
		return observeotel.StartCollector(&wg)
	},
}

func init() {
	rootCmd.AddCommand(startCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// startCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// startCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
