/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package initconfig

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/observeinc/observe-agent/internal/config"
	"github.com/observeinc/observe-agent/internal/root"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var config_path string

// flagDefaults overrides the zero-value default for fields that have no
// `default` struct tag but need a non-zero default for backward compatibility.
// Entries here are applied both to the cobra flag (shown in --help) and to
// viper (so AgentConfigFromViper returns the correct value when the flag is
// absent). Do NOT add new entries here; instead add a `default` struct tag to
// configschema.go when adding new fields (unless that would be a breaking
// change for existing config files).
var flagDefaults = map[string]any{
	"self_monitoring::enabled":                true,
	"host_monitoring::enabled":                true,
	"host_monitoring::logs::enabled":          true,
	"host_monitoring::metrics::host::enabled": true,
}

// flagDescriptions provides custom help text for specific flags. When present,
// the description is appended to the standard "Set <key>" prefix with " - ".
var flagDescriptions = map[string]string{
	"token":                                           "Observe ingest token",
	"observe_url":                                     "Observe data collection url",
	"cloud_resource_detectors":                        "The cloud environments from which to detect resources",
	"resource_attributes":                             "Attributes about the monitored host to apply to all signals",
	"application::RED_metrics::enabled":               "Enable RED metrics generation for application traces",
	"forwarding::metrics::output_format":              "Format for sending app metrics to Observe, valid options are 'prometheus' and 'otel'",
	"self_monitoring::enabled":                        "Enable self monitoring",
	"self_monitoring::fleet::enabled":                 "Enable fleet heartbeat",
	"self_monitoring::fleet::interval":                "Fleet heartbeat interval (e.g., '5m', '1h')",
	"self_monitoring::fleet::config_interval":         "Fleet config heartbeat interval (e.g., '30m', '24h')",
	"host_monitoring::enabled":                        "Enable host monitoring",
	"host_monitoring::logs::enabled":                  "Enable host monitoring logs",
	"host_monitoring::logs::include":                  "Host log file paths to include",
	"host_monitoring::logs::auto_multiline_detection": "Enable host monitoring log auto multiline detection",
	"host_monitoring::metrics::host::enabled":         "Enable host monitoring host metrics",
	"host_monitoring::metrics::process::enabled":      "Enable host monitoring process metrics",
}

func flagUsage(viperKey string) string {
	base := "Set " + viperKey
	if desc, ok := flagDescriptions[viperKey]; ok {
		return base + " - " + desc
	}
	return base
}

func NewConfigureCmd(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "init-config",
		Short: "Initialize agent configuration",
		Long:  `This command takes in parameters and creates an initialized observe agent configuration file. Will overwrite existing config file and should only be used to initialize.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var f *os.File
			if v.GetBool("print") {
				f = os.Stdout
			} else {
				var outputPath string
				if config_path != "" {
					outputPath = config_path
				} else {
					outputPath = v.ConfigFileUsed()
				}
				var err error
				f, err = os.Create(outputPath)
				if err != nil {
					return err
				}
				defer f.Close()
				fmt.Printf("Writing configuration values to %s...\n\n", outputPath)
			}
			agentConfig, err := config.AgentConfigFromViper(v)
			if err != nil {
				return err
			}
			writeConfigFile(f, agentConfig, v.GetBool("include-defaults"))
			return nil
		},
	}
}

func init() {
	v := viper.GetViper()
	configureCmd := NewConfigureCmd(v)
	RegisterConfigFlags(configureCmd, v)
	root.RootCmd.AddCommand(configureCmd)
}

func RegisterConfigFlags(cmd *cobra.Command, v *viper.Viper) {
	// Output-control flags (not part of AgentConfig).
	cmd.Flags().StringVarP(&config_path, "config_path", "", "", "Path to write config output file to")
	cmd.Flags().Bool("print", false, "Print the configuration to stdout instead of writing to a file")
	v.BindPFlag("print", cmd.Flags().Lookup("print"))
	cmd.Flags().Bool("include-defaults", false, "Include the names and default values for unset config options.")
	v.BindPFlag("include-defaults", cmd.Flags().Lookup("include-defaults"))

	// Apply viper defaults for fields that have no `default` struct tag but
	// need a non-zero default for backward compatibility.
	for key, val := range flagDefaults {
		v.SetDefault(key, val)
	}

	// Register a flag for every configurable leaf field in AgentConfig.
	skip := map[string]bool{
		"debug": true, // deprecated
	}
	registerStructFlags(cmd, v, reflect.TypeOf(config.AgentConfig{}), "", skip)
}

// registerStructFlags walks t recursively and registers a cobra persistent flag
// and viper binding for every leaf field, using the mapstructure tag as the key.
// prefix is the accumulated viper key path ("::" separated).
// skip is a set of viper keys to omit.
func registerStructFlags(cmd *cobra.Command, v *viper.Viper, t reflect.Type, prefix string, skip map[string]bool) {
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)

		mapKey := strings.Split(field.Tag.Get("mapstructure"), ",")[0]
		if mapKey == "" || mapKey == "-" {
			continue
		}

		viperKey := mapKey
		if prefix != "" {
			viperKey = prefix + "::" + mapKey
		}

		if skip[viperKey] {
			continue
		}

		defaultTag := field.Tag.Get("default")
		usage := flagUsage(viperKey)

		switch field.Type.Kind() {
		case reflect.Struct:
			registerStructFlags(cmd, v, field.Type, viperKey, skip)

		case reflect.Bool:
			defVal := resolveBoolDefault(viperKey, defaultTag)
			p := new(bool)
			cmd.PersistentFlags().BoolVar(p, viperKey, defVal, usage)
			v.BindPFlag(viperKey, cmd.PersistentFlags().Lookup(viperKey))

		case reflect.String:
			defVal := resolveStringDefault(viperKey, defaultTag)
			p := new(string)
			cmd.PersistentFlags().StringVar(p, viperKey, defVal, usage)
			v.BindPFlag(viperKey, cmd.PersistentFlags().Lookup(viperKey))

		case reflect.Int:
			defVal := resolveIntDefault(viperKey, defaultTag)
			p := new(int)
			cmd.PersistentFlags().IntVar(p, viperKey, defVal, usage)
			v.BindPFlag(viperKey, cmd.PersistentFlags().Lookup(viperKey))

		case reflect.Slice:
			if field.Type.Elem().Kind() == reflect.String {
				p := new([]string)
				cmd.PersistentFlags().StringSliceVar(p, viperKey, parseStringSliceDefault(defaultTag), usage)
				v.BindPFlag(viperKey, cmd.PersistentFlags().Lookup(viperKey))
			}
			// Non-string slices are not present in the current schema; skip.

		case reflect.Map:
			if field.Type.Key().Kind() == reflect.String &&
				field.Type.Elem().Kind() == reflect.String {
				p := &map[string]string{}
				cmd.PersistentFlags().StringToStringVar(p, viperKey, *p, usage)
				v.BindPFlag(viperKey, cmd.PersistentFlags().Lookup(viperKey))
			}
			// map[string]any (otel_config_overrides) falls here and is skipped.
		}
	}
}

func resolveBoolDefault(viperKey, defaultTag string) bool {
	if override, ok := flagDefaults[viperKey]; ok {
		if b, ok := override.(bool); ok {
			return b
		}
	}
	val, _ := strconv.ParseBool(defaultTag)
	return val
}

func resolveStringDefault(viperKey, defaultTag string) string {
	if override, ok := flagDefaults[viperKey]; ok {
		if s, ok := override.(string); ok {
			return s
		}
	}
	return defaultTag
}

func resolveIntDefault(viperKey, defaultTag string) int {
	if override, ok := flagDefaults[viperKey]; ok {
		if n, ok := override.(int); ok {
			return n
		}
	}
	val, _ := strconv.Atoi(defaultTag)
	return val
}

func parseStringSliceDefault(s string) []string {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(strings.TrimPrefix(s, "["), "]")
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
