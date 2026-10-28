// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

// Package filterottl holds the span and span event helpers that the ottl_condition policy uses,
// copied from opentelemetry-collector-contrib's internal/filter/filterottl, which can't be
// imported from outside the contrib module.
package filterottl // import "github.com/observeinc/observe-agent/components/processors/tailsamplinglegacyprocessor/internal/filterottl"

import (
	"go.opentelemetry.io/collector/component"

	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspan"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/contexts/ottlspanevent"
	"github.com/open-telemetry/opentelemetry-collector-contrib/pkg/ottl/ottlfuncs"
)

// NewBoolExprForSpan returns a ConditionSequence that is true if any of the given OTTL conditions is
// true for a span.
func NewBoolExprForSpan(conditions []string, functions map[string]ottl.Factory[*ottlspan.TransformContext], errorMode ottl.ErrorMode, set component.TelemetrySettings) (*ottl.ConditionSequence[*ottlspan.TransformContext], error) {
	parser, err := ottlspan.NewParser(functions, set)
	if err != nil {
		return nil, err
	}
	statements, err := parser.ParseConditions(conditions)
	if err != nil {
		return nil, err
	}
	c := ottlspan.NewConditionSequence(statements, set, ottlspan.WithConditionSequenceErrorMode(errorMode))
	return &c, nil
}

// NewBoolExprForSpanEvent returns a ConditionSequence that is true if any of the given OTTL
// conditions is true for a span event.
func NewBoolExprForSpanEvent(conditions []string, functions map[string]ottl.Factory[*ottlspanevent.TransformContext], errorMode ottl.ErrorMode, set component.TelemetrySettings) (*ottl.ConditionSequence[*ottlspanevent.TransformContext], error) {
	parser, err := ottlspanevent.NewParser(functions, set)
	if err != nil {
		return nil, err
	}
	statements, err := parser.ParseConditions(conditions)
	if err != nil {
		return nil, err
	}
	c := ottlspanevent.NewConditionSequence(statements, set, ottlspanevent.WithConditionSequenceErrorMode(errorMode))
	return &c, nil
}

func StandardSpanFuncs() map[string]ottl.Factory[*ottlspan.TransformContext] {
	m := ottlfuncs.StandardConverters[*ottlspan.TransformContext]()
	isRootSpanFactory := ottlfuncs.NewIsRootSpanFactory()
	m[isRootSpanFactory.Name()] = isRootSpanFactory
	return m
}

func StandardSpanEventFuncs() map[string]ottl.Factory[*ottlspanevent.TransformContext] {
	return ottlfuncs.StandardConverters[*ottlspanevent.TransformContext]()
}
