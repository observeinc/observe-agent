/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"observe/agent/cmd"
	_ "observe/agent/cmd/commands/start"
	_ "observe/agent/cmd/commands/status"
)

func main() {
	cmd.Execute()
}
