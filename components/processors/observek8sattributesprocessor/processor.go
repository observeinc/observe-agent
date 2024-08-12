package observek8sattributesprocessor

import (
	"context"
	"encoding/json"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
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
	ValueFn  func(plog.LogRecord) any
	FilterFn func(K8sEvent) bool
}

func newK8sEventsProcessor(logger *zap.Logger, cfg component.Config) *K8sEventsProcessor {
	return &K8sEventsProcessor{
		cfg:    cfg,
		logger: logger,
		actions: []K8sEventProcessorAction{
			PodStatusAction, NodeStatusAction, NodeRolesAction,
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
				bodyStr := lr.Body().AsString()
				err := json.Unmarshal([]byte(bodyStr), &event)
				if err != nil {
					kep.logger.Error("failed to unmarshal event", zap.Error(err))
					continue
				}
				var transformMap pcommon.Map
				var facetsMap pcommon.Map
				for _, action := range kep.actions {
					if action.FilterFn != nil && !action.FilterFn(event) {
						continue
					}
					transform, exists := lr.Attributes().Get("observe_transform")
					if exists {
						transformMap = transform.Map()
					} else {
						transformMap = lr.Attributes().PutEmptyMap("observe_transform")
					}
					facets, exists := transformMap.Get("facets")
					if exists {
						facetsMap = facets.Map()
					} else {
						facetsMap = transformMap.PutEmptyMap("facets")
					}

					// This is where the custom processor actually computes the value
					value := action.ValueFn(lr)

					// TODO [eg] we probably want to make this more modular. For
					// now using FromRaw for complex types is fine, since we
					// don't plan to generate arbitrarily complex/nested facets.
					// For some time coming we will produce facets that are at
					// most slices of simple types of map of simple types,
					// nothing beyond that.
					switch typed := value.(type) {
					case string:
						facetsMap.PutStr(action.Key, typed)
					case int64:
						facetsMap.PutInt(action.Key, typed)
					case bool:
						facetsMap.PutBool(action.Key, typed)
					case float64:
						facetsMap.PutDouble(action.Key, typed)
					case []string:
						// []string can't fallback to using FromRaw([]any), as
						// the default implementation of FromRaw is not smart
						// enough to understand that the slice contains all
						// string, and inserts them as bytes, instead
						slc := facetsMap.PutEmptySlice(action.Key)
						slc.EnsureCapacity(len(typed))
						for _, str := range typed {
							slc.AppendEmpty().SetStr(str)
						}
					case map[string]string:
						// Same reasoning as []string
						mp := facetsMap.PutEmptyMap(action.Key)
						mp.EnsureCapacity(len(typed))
						for k, v := range typed {
							mp.PutStr(k, v)
						}
					default:
						kep.logger.Error("sending the generated facet to Observe in bytes since no custom serialization logic is implemented", zap.Error(err))
					}
				}
			}
		}
	}
	return logs, nil
}
