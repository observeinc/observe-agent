/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package initconfig

import (
	"embed"
	"fmt"
	"html/template"
	"observe/agent/cmd"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	token       string
	observe_url string
	//go:embed observe-agent.tmpl
	configTemplateFS embed.FS
)

const configTemplate = "observe-agent.tmpl"

type AgentConfig struct {
	Token      string `yaml:"token"`
	ObserveURL string `yaml:"observe_url"`
}

var configureCmd = &cobra.Command{
	Use:   "init-config",
	Short: "Initialize agent configuration",
	Long:  `This command takes in parameters and creates an initialized observe agent configuration file. Will overwrite existing config files with default values.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		configValues := AgentConfig{
			Token:      viper.GetString("token"),
			ObserveURL: viper.GetString("observe_url"),
		}
		f, err := os.Create(viper.ConfigFileUsed())
		if err != nil {
			return err
		}
		defer f.Close()
		t := template.Must(template.New(configTemplate).ParseFS(configTemplateFS, configTemplate))
		if err := t.ExecuteTemplate(f, configTemplate, configValues); err != nil {
			return err
		}
		fmt.Print("Writing configuration values...\n\n")
		return nil
	},
}

func init() {
	configureCmd.PersistentFlags().StringVar(&token, "token", "", "Observe token")
	configureCmd.PersistentFlags().StringVar(&observe_url, "observe_url", "", "Observe data collection url")
	cmd.RootCmd.AddCommand(configureCmd)
	viper.BindPFlag("token", configureCmd.PersistentFlags().Lookup("token"))
	viper.BindPFlag("observe_url", configureCmd.PersistentFlags().Lookup("observe_url"))
}
