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
)

type Diagnostic struct {
	check        func() (any, error)
	checkName    string
	templateName string
	templateFS   embed.FS
}

var diagnostics = []Diagnostic{
	authDiagnostic(),
}

// diagnoseCmd represents the diagnose command
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Run diagnostic checks.",
	Long: `This command runs diagnostic checks for various settings and configurations
to attempt to identify issues that could cause the agent to function improperly.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Running diagnosis checks...\n")
		for _, diagnostic := range diagnostics {
			fmt.Printf("\n%s\n================\n\n", diagnostic.checkName)
			data, err := diagnostic.check()
			if err != nil {
				fmt.Printf("⚠️ Failed to run check: %s\n", err.Error())
				continue
			}
			t := template.Must(template.
				New(diagnostic.templateName).
				ParseFS(diagnostic.templateFS, diagnostic.templateName))
			if err := t.ExecuteTemplate(os.Stdout, diagnostic.templateName, data); err != nil {
				fmt.Printf("⚠️ Failed to print output for check: %s\n", err.Error())
				continue
			}
		}
	},
}

func init() {
	root.RootCmd.AddCommand(diagnoseCmd)
}
