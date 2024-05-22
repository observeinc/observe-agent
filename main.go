/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"observe/agent/cmd"
	_ "observe/agent/cmd/commands/configure"
	_ "observe/agent/cmd/commands/diagnose"
	_ "observe/agent/cmd/commands/start"
	_ "observe/agent/cmd/commands/status"
	_ "observe/agent/cmd/commands/version"
)

func runInteractive() error {
	cmd.Execute()

	return nil
}

func main() {
	run()
}
