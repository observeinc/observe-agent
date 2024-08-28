package observek8sattributesprocessor

import (
	"context"
	"encoding/json"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	EventKindPod  = "Pod"
	EventKindNode = "Node"
	EventKindJob  = "Job"
)

type K8sEventsProcessor struct {
	cfg         component.Config
	logger      *zap.Logger
	nodeActions []nodeAction
	podActions  []podAction
	jobActions  []jobAction
}

func newK8sEventsProcessor(logger *zap.Logger, cfg component.Config) *K8sEventsProcessor {
	return &K8sEventsProcessor{
		cfg:    cfg,
		logger: logger,
		podActions: []podAction{
			NewPodStatusAction(), NewPodContainersCountsAction(), NewPodReadinessAction(), NewPodConditionsAction(),
		},
		nodeActions: []nodeAction{
			NewNodeStatusAction(), NewNodeRolesAction(), NewNodePoolAction(),
		},
		jobActions: []jobAction{
			NewJobStatusAction(),
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

// Unmarshals a LogRecord into either a Node or Pod object.
func (kep *K8sEventsProcessor) unmarshalEvent(lr plog.LogRecord) metav1.Object {
	// Get the event type by unmarshalling it selectively
	var event K8sEvent
	bodyStr := lr.Body().AsString()
	err := json.Unmarshal([]byte(bodyStr), &event)
	if err != nil {
		kep.logger.Error("failed to unmarshal event", zap.Error(err))
		return nil
	}
	switch event.Kind {
	case EventKindNode:
		var node corev1.Node
		err := json.Unmarshal([]byte(lr.Body().AsString()), &node)
		if err != nil {
			kep.logger.Error("failed to unmarshal Node event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &node
	case EventKindPod:
		var pod corev1.Pod
		err := json.Unmarshal([]byte(lr.Body().AsString()), &pod)
		if err != nil {
			kep.logger.Error("failed to unmarshal Pod event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &pod
	case EventKindJob:
		var job batchv1.Job
		err := json.Unmarshal([]byte(lr.Body().AsString()), &job)
		if err != nil {
			kep.logger.Error("failed to unmarshal Job event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &job
	default:
		return nil
	}
}

func (kep *K8sEventsProcessor) processLogs(_ context.Context, logs plog.Logs) (plog.Logs, error) {
	rls := logs.ResourceLogs()
	for i := 0; i < rls.Len(); i++ {
		sls := rls.At(i).ScopeLogs()
		for j := 0; j < sls.Len(); j++ {
			lrs := sls.At(j).LogRecords()
			for k := 0; k < lrs.Len(); k++ {
				lr := lrs.At(k)
				var transformMap pcommon.Map
				var facetsMap pcommon.Map

				object := kep.unmarshalEvent(lr)
				if object == nil {
					// unmarshalEven would have already logged the error
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
					// Make sure we have capacity for at least as many actions as we have defined
					// Actions could generate more than one facet, that's taken care of afterwards.
					facetsMap.EnsureCapacity(len(kep.podActions))
				}

				// This is where the custom processor actually computes the transformed value(s)
				values, err := kep.RunActions(object)
				if err != nil {
					kep.logger.Error("could not compute attributes", zap.Error(err))
					continue
				}

				facetsMap.EnsureCapacity(facetsMap.Len() + len(values))
				for key, val := range values {
					if err := mapPut(facetsMap, key, val); err != nil {
						kep.logger.Error("could not write attributes", zap.Error(err))
						continue
					}
				}
			}
		}
	}
	return logs, nil
}

func slicePut(theSlice pcommon.Slice, value any) error {
	elem := theSlice.AppendEmpty()
	switch typed := value.(type) {
	case string:
		elem.SetStr(typed)
	case int64:
		elem.SetInt(typed)
	case bool:
		elem.SetBool(typed)
	case float64:
		elem.SetDouble(typed)
	// Let's not complicate things and avoid putting maps/slices into slices.
	// There's gotta be an easier way to model the processor's output to avoid it
	default:
		return errors.New("unrecognised type. Cannot be added to a slice")
	}

	return nil
}

// puts "anything" into a map, with some assumptions and intentional
// limitations:
//
//   - No nested slices: can only put "base types" inside slices (although
//     elements of a slice can be of different [base] types).
//
//   - Not all "base types" are covered. For instance, numbers are only int64 and float64.
//
//   - No maps with keys of arbitrary types: only string
func mapPut(theMap pcommon.Map, key string, value any) error {
	switch typed := value.(type) {
	case string:
		theMap.PutStr(key, typed)
	case int:
		theMap.PutInt(key, int64(typed))
	case int64:
		theMap.PutInt(key, typed)
	case bool:
		theMap.PutBool(key, typed)
	case float64:
		theMap.PutDouble(key, typed)
	case []any:
		slc := theMap.PutEmptySlice(key)
		slc.EnsureCapacity(len(typed))
		for _, elem := range typed {
			slicePut(slc, elem)
		}
	case attributes:
		// This is potentially arbitrarily recursive. We don't care about
		// checking the nesting level since we will never need to define
		// processors with more than one nesting level
		new := theMap.PutEmptyMap(key)
		new.EnsureCapacity(len(typed))
		for k, v := range typed {
			mapPut(new, k, v)
		}
	default:
		return errors.New("unrecognised type. Cannot be put into a map")
	}

	return nil

}
