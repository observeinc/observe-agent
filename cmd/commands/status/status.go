/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package status

import (
	"embed"
	"html/template"
	"observe/agent/cmd"
	"os"

	"github.com/spf13/cobra"
)

const statusTemplate = "status.tmpl"

var (
	//go:embed status.tmpl
	statusTemplateFS embed.FS
)

// statusCmd represents the status command
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Display status of agent",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		getStatusFromTemplate()
	},
}

func init() {
	cmd.RootCmd.AddCommand(statusCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// statusCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// statusCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func getStatusFromTemplate() error {
	data, err := GetStatusData()
	if err != nil {
		return err
	}
	t := template.Must(template.New(statusTemplate).ParseFS(statusTemplateFS, statusTemplate))
	if err := t.ExecuteTemplate(os.Stdout, statusTemplate, data); err != nil {
		return err
	}
	return nil
}
