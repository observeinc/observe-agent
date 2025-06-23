package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/invopop/jsonschema"
	"github.com/observeinc/observe-agent/internal/config"
	strcase "github.com/stoewer/go-strcase"
)

func main() {
	r := new(jsonschema.Reflector)
	r.KeyNamer = strcase.SnakeCase // from package github.com/stoewer/go-strcase
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
