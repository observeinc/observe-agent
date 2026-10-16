// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !linux

package subprocess // import "github.com/observeinc/observe-agent/components/receivers/jmxreceiver/internal/subprocess"

import (
	"os/exec"
)

func applyOSSpecificCmdModifications(_ *exec.Cmd) {}
