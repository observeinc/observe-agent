//go:build windows

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/observeinc/observe-agent/internal/root"
	"github.com/observeinc/observe-agent/observecol"
	"go.opentelemetry.io/collector/otelcol"
	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/svc"
)

func run() error {
	// If we're not running as a service, we run normally.
	if inService, err := svc.IsWindowsService(); err != nil {
		log.Fatalf("failed to determine if we are running in service: %v", err)
	} else if !inService {
		return runInteractive()
	}

	if len(os.Args) != 2 {
		log.Fatal("Expected to run svc as: observe-agent.exe <path to observe-agent.yaml>")
	}
	root.CfgFile = os.Args[1]
	root.InitConfig()

	// Get the collector settings along with our bundled config files.
	colSettings, cleanup, err := observecol.GetOtelCollectorSettings(context.Background())
	if cleanup != nil {
		defer cleanup()
	}
	if err != nil {
		return err
	}

	if err := svc.Run("", otelcol.NewSvcHandler(*colSettings)); err != nil {
		if errors.Is(err, windows.ERROR_FAILED_SERVICE_CONTROLLER_CONNECT) {
			// Per https://learn.microsoft.com/en-us/windows/win32/api/winsvc/nf-winsvc-startservicectrldispatchera#return-value
			// this means that the process is not running as a service, so run interactively.
			return runInteractive()
		}

		return fmt.Errorf("failed to start collector server: %w", err)
	}

	return nil
}
