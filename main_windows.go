//go:build windows

package main

import (
	"errors"
	"fmt"
	"log"
	"os"

	"go.opentelemetry.io/collector/otelcol"
	"golang.org/x/sys/windows"

	"observe-agent/cmd"
	"observe-agent/cmd/commands/start"

	"golang.org/x/sys/windows/svc"
)

func run() error {
	inService, err := svc.IsWindowsService()
	if err != nil {
		log.Fatalf("failed to determine if we are running in service: %v", err)
	}

	if inService {
		if len(os.Args) != 2 {
			log.Fatal("Expected to run svc as: observe-agent.exe <path to observe-agent.yaml>")
		}
		cmd.CfgFile = os.Args[1]
		cmd.InitConfig()
		colSettings, cleanup, err := start.SetupAndGenerateCollectorSettings()
		if err != nil {
			return err
		}
		if cleanup != nil {
			defer cleanup()
		}
		if err := svc.Run("", otelcol.NewSvcHandler(*colSettings)); err != nil {
			if errors.Is(err, windows.ERROR_FAILED_SERVICE_CONTROLLER_CONNECT) {
				// Per https://learn.microsoft.com/en-us/windows/win32/api/winsvc/nf-winsvc-startservicectrldispatchera#return-value
				// this means that the process is not running as a service, so run interactively.
				return runInteractive()
			}

			return fmt.Errorf("failed to start collector server: %w", err)
		}
	} else {
		return runInteractive()
	}

	return nil
}
