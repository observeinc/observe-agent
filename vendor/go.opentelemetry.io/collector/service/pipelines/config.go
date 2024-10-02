// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package pipelines // import "go.opentelemetry.io/collector/service/pipelines"

import (
	"errors"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componentprofiles"
	"go.opentelemetry.io/collector/pipeline"
)

var (
	errMissingServicePipelines         = errors.New("service must have at least one pipeline")
	errMissingServicePipelineReceivers = errors.New("must have at least one receiver")
	errMissingServicePipelineExporters = errors.New("must have at least one exporter")
)

// Config defines the configurable settings for service telemetry.
//
// Deprecated: [v0.110.0] Use ConfigWithPipelineID instead
type Config map[component.ID]*PipelineConfig

func (cfg Config) Validate() error {
	// Must have at least one pipeline.
	if len(cfg) == 0 {
		return errMissingServicePipelines
	}

	// Check that all pipelines have at least one receiver and one exporter, and they reference
	// only configured components.
	for pipelineID, p := range cfg {
		switch pipelineID.Type() {
		// nolint
		case component.DataTypeTraces, component.DataTypeMetrics, component.DataTypeLogs, componentprofiles.DataTypeProfiles:
			// Continue
		default:
			return fmt.Errorf("pipeline %q: unknown datatype %q", pipelineID, pipelineID.Type())
		}

		// Validate pipeline has at least one receiver.
		if err := p.Validate(); err != nil {
			return fmt.Errorf("pipeline %q: %w", pipelineID, err)
		}
	}

	return nil
}

type ConfigWithPipelineID map[pipeline.ID]*PipelineConfig

func (cfg ConfigWithPipelineID) Validate() error {
	// Must have at least one pipeline.
	if len(cfg) == 0 {
		return errMissingServicePipelines
	}

	// Check that all pipelines have at least one receiver and one exporter, and they reference
	// only configured components.
	for pipelineID, p := range cfg {
		switch pipelineID.Signal() {
		case pipeline.SignalTraces, pipeline.SignalMetrics, pipeline.SignalLogs, componentprofiles.SignalProfiles:
			// Continue
		default:
			return fmt.Errorf("pipeline %q: unknown signal %q", pipelineID.String(), pipelineID.Signal())
		}

		// Validate pipeline has at least one receiver.
		if err := p.Validate(); err != nil {
			return fmt.Errorf("pipeline %q: %w", pipelineID.String(), err)
		}
	}

	return nil
}

// PipelineConfig defines the configuration of a Pipeline.
type PipelineConfig struct {
	Receivers  []component.ID `mapstructure:"receivers"`
	Processors []component.ID `mapstructure:"processors"`
	Exporters  []component.ID `mapstructure:"exporters"`
}

func (cfg *PipelineConfig) Validate() error {
	// Validate pipeline has at least one receiver.
	if len(cfg.Receivers) == 0 {
		return errMissingServicePipelineReceivers
	}

	// Validate pipeline has at least one exporter.
	if len(cfg.Exporters) == 0 {
		return errMissingServicePipelineExporters
	}

	// Validate no processors are duplicated within a pipeline.
	procSet := make(map[component.ID]struct{}, len(cfg.Processors))
	for _, ref := range cfg.Processors {
		// Ensure no processors are duplicated within the pipeline
		if _, exists := procSet[ref]; exists {
			return fmt.Errorf("references processor %q multiple times", ref)
		}
		procSet[ref] = struct{}{}
	}

	return nil
}
