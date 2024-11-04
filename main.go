/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	_ "github.com/observeinc/observe-agent/internal/commands/config"
	_ "github.com/observeinc/observe-agent/internal/commands/diagnose"
	_ "github.com/observeinc/observe-agent/internal/commands/initconfig"
	_ "github.com/observeinc/observe-agent/internal/commands/sendtestdata"
	_ "github.com/observeinc/observe-agent/internal/commands/start"
	_ "github.com/observeinc/observe-agent/internal/commands/status"
	_ "github.com/observeinc/observe-agent/internal/commands/version"
	"github.com/observeinc/observe-agent/internal/root"
)

func runInteractive() error {
	root.Execute()
	return nil
}

func main() {
	run()
}
