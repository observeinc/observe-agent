/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	_ "observe-agent/internal/commands/diagnose"
	_ "observe-agent/internal/commands/initconfig"
	_ "observe-agent/internal/commands/start"
	_ "observe-agent/internal/commands/status"
	_ "observe-agent/internal/commands/version"
	"observe-agent/internal/root"
)

func runInteractive() error {
	root.Execute()
	return nil
}

func main() {
	run()
}
