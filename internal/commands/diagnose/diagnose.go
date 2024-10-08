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

// diagnoseCmd represents the diagnose command
var diagnoseCmd = &cobra.Command{
	Use:   "diagnose",
	Short: "Run diagnostic checks.",
	Long: `This command runs diagnostic checks for various settings and configurations
to attempt to identify issues that could cause the agent to function improperly.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Print("Running diagnosis checks...\n\n")
		runNetworkCheck()
	},
}

func init() {
	root.RootCmd.AddCommand(diagnoseCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// diagnoseCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// diagnoseCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func runNetworkCheck() error {
	collector_url := viper.GetString("observe_url")
	authTestResponse := makeAuthTestRequest(collector_url)
	t := template.Must(template.New(authcheckTemplate).ParseFS(authcheckTemplateFS, authcheckTemplate))
	if err := t.ExecuteTemplate(os.Stdout, authcheckTemplate, authTestResponse); err != nil {
		return err
	}
	return nil
}
