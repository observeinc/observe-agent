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

type Diagnostic struct {
	check        func(*viper.Viper) (bool, any, error)
	checkName    string
	templateName string
	templateFS   embed.FS
}

var diagnostics = []Diagnostic{
	configDiagnostic(),
	otelconfigDiagnostic(),
	agentstatusDiagnostic(),
	authDiagnostic(),
}

// diagnoseCmd represents the diagnose command
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Run diagnostic checks.",
	Long: `This command runs diagnostic checks for various settings and configurations
to attempt to identify issues that could cause the agent to function improperly.`,
	Run: func(cmd *cobra.Command, args []string) {
		v := viper.GetViper()
		fmt.Print("Running diagnosis checks...\n")
		var failedChecks []string
		for _, diagnostic := range diagnostics {
			fmt.Printf("\n%s\n================\n\n", diagnostic.checkName)
			success, data, err := diagnostic.check(v)
			if !success {
				failedChecks = append(failedChecks, diagnostic.checkName)
			}
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
		if len(failedChecks) > 0 {
			fmt.Printf("\n❌ %d out of %d checks failed:\n", len(failedChecks), len(diagnostics))
			for _, check := range failedChecks {
				fmt.Printf("  - %s\n", check)
			}
			os.Exit(1)
		} else {
			fmt.Printf("\n✅ All %d checks passed!\n", len(diagnostics))
		}
	},
}

func init() {
	root.RootCmd.AddCommand(diagnoseCmd)
}
