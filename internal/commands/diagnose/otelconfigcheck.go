package diagnose

import (
	"context"
	"embed"

	"github.com/observeinc/observe-agent/internal/commands/start"
	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/otelcol"
)

type OtelConfigTestResult struct {
	Passed bool
	Error  string
}

func checkOtelConfig(_ *viper.Viper) (bool, any, error) {
	configFilePaths, cleanup, err := start.SetupAndGetConfigFiles(logger.WithCtx(context.Background(), logger.GetNop()))
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return false, nil, err
	}
	colSettings := observecol.GenerateCollectorSettingsWithConfigFiles(configFilePaths)
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
