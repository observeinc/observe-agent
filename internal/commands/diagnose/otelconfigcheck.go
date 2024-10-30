package diagnose

import (
	"context"
	"embed"

	"github.com/observeinc/observe-agent/internal/commands/start"
	logger "github.com/observeinc/observe-agent/internal/commands/util"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/otelcol"
)

type OtelConfigTestResult struct {
	Passed bool
	Error  string
}

func checkOtelConfig(_ *viper.Viper) (bool, any, error) {
	colSettings, cleanup, err := start.SetupAndGenerateCollectorSettings(logger.WithCtx(context.Background(), logger.GetNop()))
	if err != nil {
		return false, nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	// These are the same checks as the `otelcol validate` command:
	// https://github.com/open-telemetry/opentelemetry-collector/blob/main/otelcol/command_validate.go
	col, err := otelcol.NewCollector(*colSettings)
	if err != nil {
		return false, nil, err
	}
	err = col.DryRun(context.Background())
	if err != nil {
		return false, OtelConfigTestResult{
			Passed: false,
			Error:  err.Error(),
		}, nil
	}
	return true, OtelConfigTestResult{
		Passed: true,
	}, nil
}

const otelconfigcheckTemplate = "otelconfigcheck.tmpl"

var (
	//go:embed otelconfigcheck.tmpl
	otelconfigcheckTemplateFS embed.FS
)

func otelconfigDiagnostic() Diagnostic {
	return Diagnostic{
		check:        checkOtelConfig,
		checkName:    "OTEL Config Check",
		templateName: otelconfigcheckTemplate,
		templateFS:   otelconfigcheckTemplateFS,
	}
}
