package observek8sattributesprocessor

import (
	// "github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/attraction"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
)

type k8sEventsProcessor struct {
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

func newK8sEventsProcessor() (*k8sEventsProcessor, error) {
	return &k8sEventsProcessor{
		logger: zap.NewNop(),
	}, nil
}

func (kep *k8sEventsProcessor) processLogs(logs plog.Logs) (plog.Logs, error) {
	rl := logs.ResourceLogs()
	for i := 0; i < rl.Len(); i++ {
		sl := rl.At(i).ScopeLogs()
		for j := 0; j < sl.Len(); j++ {
			lr := sl.At(j).LogRecords()
			for k := 0; k < lr.Len(); k++ {
				lr := lr.At(k)
				lr.Attributes().PutStr(PodStatusAttributeKey, getStatus(lr))
			}
		}
	}
	return logs, nil
}
