/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package status

import (
	"embed"
	"html/template"
	"os"

	"github.com/observeinc/observe-agent/internal/connections"
	"github.com/observeinc/observe-agent/internal/root"
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
	root.RootCmd.AddCommand(statusCmd)
}

func getStatusFromTemplate() error {
	data, err := GetStatusData()
	if err != nil {
		return err
	}
	t := template.Must(template.New(statusTemplate).
		Funcs(connections.GetTemplateFuncMap()).
		ParseFS(statusTemplateFS, statusTemplate))
	if err := t.ExecuteTemplate(os.Stdout, statusTemplate, data); err != nil {
		return err
	}
	return nil
}
