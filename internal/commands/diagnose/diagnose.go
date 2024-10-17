/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package diagnose

import (
	"embed"
	"fmt"
	"os"
	"text/template"

	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// const networkcheckTemplate = "networkcheck.tmpl"
const authcheckTemplate = "authcheck.tmpl"

var (
	//go:embed authcheck.tmpl
	authcheckTemplateFS embed.FS
)

var diagnostics = []struct {
	check        func() (any, error)
	templateName string
	templateFS   embed.FS
}{
	{
		authCheck,
		authcheckTemplate,
		authcheckTemplateFS,
	},
}

// diagnoseCmd represents the diagnose command
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Run diagnostic checks.",
	Long: `This command runs diagnostic checks for various settings and configurations
to attempt to identify issues that could cause the agent to function improperly.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Print("Running diagnosis checks...\n\n")
		for _, diagnostic := range diagnostics {
			data, err := diagnostic.check()
			if err != nil {
				return err
			}
			t := template.Must(template.
				New(diagnostic.templateName).
				ParseFS(diagnostic.templateFS, diagnostic.templateName))
			if err := t.ExecuteTemplate(os.Stdout, diagnostic.templateName, data); err != nil {
				return err
			}
		}
		return nil
	},
}

func init() {
	root.RootCmd.AddCommand(diagnoseCmd)
}

func authCheck() (any, error) {
	collector_url := viper.GetString("observe_url")
	authTestResponse := makeAuthTestRequest(collector_url)
	return authTestResponse, nil
}
