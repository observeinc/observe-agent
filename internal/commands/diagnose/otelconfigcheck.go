package diagnose

import (
	"context"
	"embed"

	"github.com/observeinc/observe-agent/internal/commands/start"
	"github.com/spf13/viper"
	"go.opentelemetry.io/collector/otelcol"
)

type OtelConfigTestResult struct {
	Passed bool
	Error  string
}

func checkOtelConfig(_ *viper.Viper) (any, error) {
	colSettings, cleanup, err := start.SetupAndGenerateCollectorSettings()
	if err != nil {
		return nil, err
	}
	if cleanup != nil {
		defer cleanup()
	}
	// Copying the implementation from the `otelcol validate` command
	col, err := otelcol.NewCollector(*colSettings)
	if err != nil {
		return nil, err
	}
	err = col.DryRun(context.Background())
	if err != nil {
		return OtelConfigTestResult{
			Passed: false,
			Error:  err.Error(),
		}, nil
	}
	return OtelConfigTestResult{
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
