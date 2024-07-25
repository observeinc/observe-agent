/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package version

import (
	"fmt"
	"observe-agent/build"
	"observe-agent/cmd"

	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Display the currently installed version of the observe-agent.",
	Long: `Display the currently installed version of the observe-agent. This version
is based on the package release.`,
	Run: func(cmd *cobra.Command, args []string) {
		version := getVersion()
		fmt.Printf("observe-agent version: %s\n", version)
	},
}

func init() {
	cmd.RootCmd.AddCommand(versionCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// versionCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getVersion() string {
	if build.Version == "" {
		return "dev"
	}
	return build.Version
}
