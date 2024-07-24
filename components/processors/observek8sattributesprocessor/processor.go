package observek8sattributesprocessor

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type K8sEvent struct {
	Kind       string `json:"kind,omitempty"`
	ApiVersion string `json:"apiVersion,omitempty"`
}

type K8sEventsProcessor struct {
	cfg     component.Config
	logger  *zap.Logger
	actions []K8sEventProcessorAction
}

type K8sEventProcessorAction struct {
	Key      string
	ValueFn  func(plog.LogRecord) string
	FilterFn func(K8sEvent) bool
}

func newK8sEventsProcessor(logger *zap.Logger, cfg component.Config) *K8sEventsProcessor {
	return &K8sEventsProcessor{
		cfg:    cfg,
		logger: logger,
		actions: []K8sEventProcessorAction{
			PodStatusAction,
		},
	}
}

func (kep *K8sEventsProcessor) Start(_ context.Context, _ component.Host) error {
	kep.logger.Info("observek8sattributes processor has started.")
	return nil
}

func (kep *K8sEventsProcessor) Shutdown(_ context.Context) error {
	kep.logger.Info("observek8sattributes processor shutting down.")
	return nil
}

func (kep *K8sEventsProcessor) processLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	rls := logs.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		sls := rls.At(i).ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			lrs := sls.At(j).LogRecords()
			for k := 0; k < lrs.Len(); k++ {
				lr := lrs.At(k)
				var event K8sEvent
				err := json.Unmarshal([]byte(lr.Body().AsString()), &event)
				if err != nil {
					kep.logger.Error("failed to unmarshal event", zap.Error(err))
					continue
				}
				for _, action := range kep.actions {
					if action.FilterFn != nil && !action.FilterFn(event) {
						continue
					}
					transform, exists := lr.Attributes().Get("observe_transform")
					if exists {
						facets, exists := transform.Map().Get("facets")
						if exists {
							facets.Map().PutStr(action.Key, action.ValueFn(lr))
						} else {
							facets := transform.Map().PutEmptyMap("facets")
							facets.PutStr(action.Key, action.ValueFn(lr))
						}
					} else {
						transform := lr.Attributes().PutEmptyMap("observe_transform")
						facets := transform.PutEmptyMap("facets")
						facets.PutStr(action.Key, action.ValueFn(lr))
					}
				}
			}
		}
	}
	return logs, nil
}
