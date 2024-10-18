package observek8sattributesprocessor

import (
	"context"
	"encoding/json"
	"errors"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// CORE
	EventKindPod            = "Pod"
	EventKindNode           = "Node"
	EventKindService        = "Service"
	EventKindServiceAccount = "ServiceAccount"
	EventKindEndpoints      = "Endpoints"
	EventKindConfigMap      = "ConfigMap"
	EventKindSecret         = "Secret"
	// APPS
	EventKindStatefulSet = "StatefulSet"
	EventKindDaemonSet   = "DaemonSet"
	EventKindDeployment  = "Deployment"
	// WORKLOAD
	EventKindJob     = "Job"
	EventKindCronJob = "CronJob"
	// STORAGE
	EventKindPersistentVolume      = "PersistentVolume"
	EventKindPersistentVolumeClaim = "PersistentVolumeClaim"
	// NETWORK
	EventKindIngress = "Ingress"
)

type K8sEventsProcessor struct {
	cfg    component.Config
	logger *zap.Logger

	// --------------- actions that generate new attribute(s) ---------------
	nodeActions           []nodeAction
	podActions            []podAction
	endpointsActions      []endpointsAction
	serviceActions        []serviceAction
	serviceAccountActions []serviceAccountAction
	configMapActions      []configMapAction

	jobActions     []jobAction
	cronJobActions []cronJobAction

	daemonSetActions   []daemonSetAction
	statefulSetActions []statefulSetAction
	deploymentActions  []deploymentAction

	persistentVolumeActions      []persistentVolumeAction
	persistentVolumeClaimActions []persistentVolumeClaimAction

	ingressActions []ingressAction

	// --------------- actions that edit the body of an event IN PLACE ---------------
	secretBodyActions []secretBodyAction
	// Other slices should be: <anotherObject>BodyActions []bodyAction
}

func newK8sEventsProcessor(logger *zap.Logger, cfg component.Config) *K8sEventsProcessor {
	return &K8sEventsProcessor{
		cfg:    cfg,
		logger: logger,

		// --------------- actions that generate new attribute(s) ---------------
		podActions: []podAction{
			NewPodStatusAction(), NewPodContainersCountsAction(), NewPodReadinessAction(), NewPodConditionsAction(),
		},
		nodeActions: []nodeAction{
			NewNodeStatusAction(), NewNodeRolesAction(), NewNodePoolAction(),
		},
		endpointsActions: []endpointsAction{
			NewEndpointsStatusAction(),
		},
		serviceActions: []serviceAction{
			NewServiceLBIngressAction(), NewServiceSelectorAction(), NewServicePortsAction(), NewServiceExternalIPsAction(),
		},
		serviceAccountActions: []serviceAccountAction{
			NewServiceAccountSecretsNamesAction(), NewServiceAccountSecretsAction(), NewServiceAccountImagePullSecretsAction(),
		},
		configMapActions: []configMapAction{
			NewConfigMapDataAction(),
		},

		jobActions: []jobAction{
			NewJobStatusAction(), NewJobDurationAction(),
		},
		cronJobActions: []cronJobAction{
			NewCronJobActiveAction(),
		},

		deploymentActions: []deploymentAction{
			NewDeploymentSelectorAction(),
		},
		statefulSetActions: []statefulSetAction{
			NewStatefulsetSelectorAction(),
		},
		daemonSetActions: []daemonSetAction{
			NewDaemonSetSelectorAction(),
		},

		persistentVolumeActions: []persistentVolumeAction{
			NewPersistentVolumeTypeAction(),
		},
		persistentVolumeClaimActions: []persistentVolumeClaimAction{
			NewPersistentVolumeClaimSelectorAction(),
		},

		ingressActions: []ingressAction{
			NewIngressRulesAction(), NewIngressLoadBalancerAction(),
		},

		// --------------- actions that edit the body of an event IN PLACE ---------------
		secretBodyActions: []secretBodyAction{
			NewSecretRedactorBodyAction(),
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

// Unmarshals a LogRecord into the corresponding k8s object.
// It does so by first unmarshaling the event into a minimal struct that
// contains the event type. Based on such type, it then unmarshals the whole
// event into the typed API object.
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
	case EventKindEndpoints:
		var endpoints corev1.Endpoints
		err := json.Unmarshal([]byte(lr.Body().AsString()), &endpoints)
		if err != nil {
			kep.logger.Error("failed to unmarshal Endpoints event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &endpoints
	case EventKindService:
		var service corev1.Service
		err := json.Unmarshal([]byte(lr.Body().AsString()), &service)
		if err != nil {
			kep.logger.Error("failed to unmarshal Service event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &service
	case EventKindServiceAccount:
		var sa corev1.ServiceAccount
		err := json.Unmarshal([]byte(lr.Body().AsString()), &sa)
		if err != nil {
			kep.logger.Error("failed to unmarshal ServiceAccount event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &sa
	case EventKindConfigMap:
		var configMap corev1.ConfigMap
		err := json.Unmarshal([]byte(lr.Body().AsString()), &configMap)
		if err != nil {
			kep.logger.Error("failed to unmarshal ConfigMap event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &configMap
	case EventKindSecret:
		var secret corev1.Secret
		err := json.Unmarshal([]byte(lr.Body().AsString()), &secret)
		if err != nil {
			kep.logger.Error("failed to unmarshal Secret event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &secret
	case EventKindDeployment:
		var deployment appsv1.Deployment
		err := json.Unmarshal([]byte(lr.Body().AsString()), &deployment)
		if err != nil {
			kep.logger.Error("failed to unmarshal Deployment event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &deployment
	case EventKindStatefulSet:
		var statefulSet appsv1.StatefulSet
		err := json.Unmarshal([]byte(lr.Body().AsString()), &statefulSet)
		if err != nil {
			kep.logger.Error("failed to unmarshal StatefulSet event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &statefulSet
	case EventKindDaemonSet:
		var daemonSet appsv1.DaemonSet
		err := json.Unmarshal([]byte(lr.Body().AsString()), &daemonSet)
		if err != nil {
			kep.logger.Error("failed to unmarshal DaemonSet event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &daemonSet
	case EventKindJob:
		var job batchv1.Job
		err := json.Unmarshal([]byte(lr.Body().AsString()), &job)
		if err != nil {
			kep.logger.Error("failed to unmarshal Job event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &job
	case EventKindCronJob:
		var cronJob batchv1.CronJob
		err := json.Unmarshal([]byte(lr.Body().AsString()), &cronJob)
		if err != nil {
			kep.logger.Error("failed to unmarshal CronJob event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &cronJob
	case EventKindPersistentVolume:
		var persistentVolume corev1.PersistentVolume
		err := json.Unmarshal([]byte(lr.Body().AsString()), &persistentVolume)
		if err != nil {
			kep.logger.Error("failed to unmarshal PersistentVolume event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &persistentVolume
	case EventKindPersistentVolumeClaim:
		var persistentVolumeClaim corev1.PersistentVolumeClaim
		err := json.Unmarshal([]byte(lr.Body().AsString()), &persistentVolumeClaim)
		if err != nil {
			kep.logger.Error("failed to unmarshal PersistentVolumeClaim event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &persistentVolumeClaim
	case EventKindIngress:
		var ingress netv1.Ingress
		err := json.Unmarshal([]byte(lr.Body().AsString()), &ingress)
		if err != nil {
			kep.logger.Error("failed to unmarshal Ingress event %v", zap.Error(err), zap.String("event", lr.Body().AsString()))
			return nil
		}
		return &ingress
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

				// ALWAYS RUN BODY ACTIONS FIRST
				// The attributes should always be computed on the MODIFIED body.
				err := kep.RunBodyActions(object)
				if err != nil {
					kep.logger.Error("could not run body actions", zap.Error(err))
					continue
				}
				// We now re-marshal the object
				reMarshsalledBody, err := json.Marshal(object)
				if err != nil {
					kep.logger.Error("could not re-marshal body", zap.Error(err))
					continue
				}
				// And update the body of the log record
				lr.Body().SetStr(string(reMarshsalledBody))

				// Add attributes["observe_transform"]["facets"] if it doesn't exist
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

				// Compute custom attributes
				values, err := kep.RunActions(object)
				if err != nil {
					kep.logger.Error("could not compute attributes", zap.Error(err))
					continue
				}

				// Add them to the facets
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
	case []string:
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
