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
	"github.com/spf13/viper"
)

const statusTemplate = "status.tmpl"

var (
	//go:embed status.tmpl
	statusTemplateFS embed.FS
)

func NewStatusCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Display status of agent",
		Long:  ``,
		Run: func(cmd *cobra.Command, args []string) {
			getStatusFromTemplate(v)
		},
	}
}

func init() {
	v := viper.GetViper()
	statusCmd := NewStatusCmd(v)
	RegisterStatusFlags(statusCmd, v)
	root.RootCmd.AddCommand(statusCmd)
}

func getStatusFromTemplate(v *viper.Viper) error {
	data, err := GetStatusData(v.GetString(TelemetryEndpointFlag), v.GetString(HealthcheckEndpointFlag), v.GetString(HealthcheckPathFlag))
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
