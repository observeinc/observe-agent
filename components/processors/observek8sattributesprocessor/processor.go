package observek8sattributesprocessor

import (
	// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/attraction"
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type k8sEventsProcessor struct {
	cfg    component.Config
	logger *zap.Logger
}

// func newK8sEventAttributesProcessor(logger *zap.Logger, status string) *attraction.AttrProc {
// 	actions := []attraction.Action{
// 		{
// 			Key:            "observe_transform.facets.status",
// 			AttributeValue: status,
// 			Action:         attraction.INSERT,
// 		},
// 	}
// 	return &attraction.AttrProc{actions: actions}
// }

func newK8sEventsProcessor(logger *zap.Logger, cfg component.Config) *k8sEventsProcessor {
	return &k8sEventsProcessor{
		cfg:    cfg,
		logger: logger,
	}
}

func (kep *k8sEventsProcessor) Start(_ context.Context, _ component.Host) error {
	kep.logger.Info("observek8sattributes processor has started.")
	return nil
}

func (kep *k8sEventsProcessor) Shutdown(_ context.Context) error {
	kep.logger.Info("observek8sattributes processor shutting down.")
	return nil
}

func (kep *k8sEventsProcessor) processLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	rl := logs.ResourceLogs()
	for i := 0; i < rl.Len(); i++ {
		sl := rl.At(i).ScopeLogs()
		for j := 0; j < sl.Len(); j++ {
			lr := sl.At(j).LogRecords()
			for k := 0; k < lr.Len(); k++ {
				lr := lr.At(k)
				status := getStatus(lr)
				if status != "" {
					logLine := fmt.Sprintf("calculated status: %s for event: %s", status, lr.Body().AsString())
					kep.logger.Info(logLine)
				}
				currObserveFacets, exists := lr.Attributes().Get("observe_transform")
				if exists {
					facets := currObserveFacets.Map()

					facets.PutStr(PodStatusAttributeKey, getStatus(lr))
				}
				lr.Attributes().PutStr(PodStatusAttributeKey, getStatus(lr))
			}
		}
	}
	return logs, nil
}
