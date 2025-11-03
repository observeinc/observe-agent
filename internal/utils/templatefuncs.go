package utils

import (
	"html/template"
	"maps"
	"strings"

	"github.com/Masterminds/sprig/v3"
	"github.com/goccy/go-yaml"
)

func TemplateFuncs() template.FuncMap {
	// Sprig documentation can be found here: https://masterminds.github.io/sprig/
	sprigMap := sprig.FuncMap()
	extra := template.FuncMap{
		"toYaml":     toYAML,
		"mustToYaml": mustToYAML,
	}
	maps.Copy(sprigMap, extra)
	return sprigMap
}

// ================================================================
// The following methods are all taken directly from Helm source:
// https://github.com/helm/helm/blob/bdc459d73c44c6dad06289b49803cb1b3b4e21b7/pkg/engine/funcs.go
// ================================================================

// toYAML takes an interface, marshals it to yaml, and returns a string. It will
// always return a string, even on marshal error (empty string).
//
// This is designed to be called from a template.
func toYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		// Swallow errors inside of a template.
		return ""
	}
	return strings.TrimSuffix(string(data), "\n")
}

// mustToYAML takes an interface, marshals it to yaml, and returns a string.
// It will panic if there is an error.
//
// This is designed to be called from a template when need to ensure that the
// output YAML is valid.
func mustToYAML(v interface{}) string {
	data, err := yaml.Marshal(v)
	if err != nil {
		panic(err)
	}
	return strings.TrimSuffix(string(data), "\n")
}
