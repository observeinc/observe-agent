package gocommonscripts

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Agent struct {
	agentPath  string
	configPath string
	output     string
	envVars    map[string]string
	process    *exec.Cmd
}

func NewAgent(agentPath string, configPath string, output string, envVars map[string]string) (*Agent, error) {
	if _, err := os.Stat(agentPath); err != nil {
		return nil, fmt.Errorf("agent path %s does not exist", agentPath)
	}
	if _, err := os.Stat(configPath); err != nil {
		return nil, fmt.Errorf("config path %s does not exist", configPath)
	}
	return &Agent{
		agentPath:  agentPath,
		configPath: configPath,
		output:     output,
		envVars:    envVars,
		process:    nil,
	}, nil
}

func (a *Agent) Start() error {
	a.process = nil
	a.process.Stdout = os.Stdout
	a.process.Stderr = os.Stderr
	return a.process.Start()
}

func (a *Agent) Stop() error {
	if a.process == nil {
		return nil
	}
	return a.process.Process.Signal(syscall.SIGTERM)
}

func (a *Agent) Wait() error {
	if a.process == nil {
		return nil
	}
	return a.process.Wait()
}

func (a *Agent) Restart() error {
	if err := a.Stop(); err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return a.Start()
}
