package goscripts

import (
	"testing"

	"github.com/observeinc/observe-agent/integration/gocommonscripts"
)

func TestInstall(t *testing.T) {
	t.Log("This is a test")

	// use subprocess to feed input into agent

	// path to executable, path to config, (both in repo), path to output file (somewhere on system)
	agentPath := "/path/to/agent"
	configPath := "/path/to/config"
	outputPath := "/path/to/output"

	agent, err := gocommonscripts.NewAgent(agentPath, configPath, outputPath)
	if err != nil {
		t.Fatal(err)
	}

	agent.Start()

	defer agent.Stop()

	// check if input is as desired
	// assert on a line-by-line basis? Or just check whole file matches?
}
