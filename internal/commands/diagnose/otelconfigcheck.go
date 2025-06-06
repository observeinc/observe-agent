package diagnose

import (
	"context"
	"embed"

	"github.com/observeinc/observe-agent/internal/commands/util/logger"
	"github.com/observeinc/observe-agent/observecol"
	"github.com/spf13/viper"
)

type OtelConfigTestResult struct {
	Passed bool
	Error  string
}

func checkOtelConfig(_ *viper.Viper) (bool, any, error) {
	ctx := logger.WithCtx(context.Background(), logger.GetNop())
	col, cleanup, err := observecol.GetOtelCollector(ctx)
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return false, nil, err
	}
	// These are the same checks as the `otelcol validate` command:
	// https://github.com/open-telemetry/opentelemetry-collector/blob/main/otelcol/command_validate.go
	err = col.DryRun(ctx)
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
