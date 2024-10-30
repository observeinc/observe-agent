package diagnose

import (
	"embed"

	"github.com/observeinc/observe-agent/internal/commands/status"
	"github.com/spf13/viper"
)

type StatusTestResult struct {
	Passed       bool
	AgentRunning bool
	Error        string
}

func checkStatus(_ *viper.Viper) (bool, any, error) {
	data, err := status.GetStatusData()
	if err != nil {
		return false, StatusTestResult{
			Passed:       false,
			AgentRunning: false,
			Error:        err.Error(),
		}, nil
	}
	if data.Status != status.Running.String() {
		return false, StatusTestResult{
			Passed:       false,
			AgentRunning: false,
			Error:        "agent is not running",
		}, nil
	}
	if data.AgentMetrics == (status.AgentMetrics{}) {
		return false, StatusTestResult{
			Passed:       false,
			AgentRunning: true,
			Error:        "agent metrics are not available",
		}, nil
	}
	return true, StatusTestResult{
		Passed:       true,
		AgentRunning: true,
	}, nil
}

// const networkcheckTemplate = "networkcheck.tmpl"
const agentStatusCheckTemplate = "agentstatuscheck.tmpl"

var (
	//go:embed agentstatuscheck.tmpl
	agentStatusCheckTemplateFS embed.FS
)

func agentstatusDiagnostic() Diagnostic {
	return Diagnostic{
		check:        checkStatus,
		checkName:    "Agent Status Check",
		templateName: agentStatusCheckTemplate,
		templateFS:   agentStatusCheckTemplateFS,
	}
}
