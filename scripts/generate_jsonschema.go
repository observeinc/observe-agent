package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/observeinc/observe-agent/internal/config"
	strcase "github.com/stoewer/go-strcase"
)

func main() {
	r := new(jsonschema.Reflector)
	r.KeyNamer = func(name string) string {
		// If the name already contains an underscore, assume it's a custom name.
		if strings.Contains(name, "_") {
			return name
		}
		// from package github.com/stoewer/go-strcase
		return strcase.SnakeCase(name)
	}
	r.RequiredFromJSONSchemaTags = true

	s := r.Reflect(&config.AgentConfig{})

	schema, err := s.MarshalJSON()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling schema: %v\n", err)
		os.Exit(1)
	}

	// Parse and pretty-print the JSON
	var prettyJSON interface{}
	if err := json.Unmarshal(schema, &prettyJSON); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing schema JSON: %v\n", err)
		os.Exit(1)
	}

	prettySchema, err := json.MarshalIndent(prettyJSON, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error pretty-printing schema: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	if err := os.WriteFile("observe-agent.schema.json", prettySchema, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing schema to file: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("JSON schema written to observe-agent.schema.json")
}
