package observek8sattributesprocessor

import (
	"sort"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	netv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type K8sEvent struct {
	Kind       string `json:"kind,omitempty"`
	ApiVersion string `json:"apiVersion,omitempty"`
}

type attributes map[string]any

func (atts attributes) addAttributes(other attributes) {
	for k, v := range other {
		atts[k] = v
	}
}

type Set map[string]string

// FormatLabels converts label map into plain string
func FormatLabels(labelMap map[string]string) string {
	l := Set(labelMap).String()
	if l == "" {
		l = "<none>"
	}
	return l
}

// String returns all labels listed as a human readable string.
// Conveniently, exactly the format that ParseSelector takes.
func (ls Set) String() string {
	selector := make([]string, 0, len(ls))
	for key, value := range ls {
		selector = append(selector, key+"="+value)
	}
	// Sort for determinism.
	sort.StringSlice(selector).Sort()
	return strings.Join(selector, ",")
}

// Action that processes a K8S object and computes custom attributes for it
type K8sEventProcessorAction interface {
	// Computes attributes for a k8s entity.  Since entities like Pod, Node,
	// etc. don't have a common interface, the argument of ComputeAttributes is
	// of type any types that implement this method should check that the arg is
	// of the right type before proceeding.
	// Check the utility functions below for more info
	ComputeAttributes(any) (attributes, error)
}

type podAction interface {
	ComputeAttributes(corev1.Pod) (attributes, error)
}
type nodeAction interface {
	ComputeAttributes(corev1.Node) (attributes, error)
}
type serviceAction interface {
	ComputeAttributes(corev1.Service) (attributes, error)
}
type serviceAccountAction interface {
	ComputeAttributes(corev1.ServiceAccount) (attributes, error)
}

type jobAction interface {
	ComputeAttributes(batchv1.Job) (attributes, error)
}
type cronJobAction interface {
	ComputeAttributes(batchv1.CronJob) (attributes, error)
}

type statefulSetAction interface {
	ComputeAttributes(appsv1.StatefulSet) (attributes, error)
}
type daemonSetAction interface {
	ComputeAttributes(appsv1.DaemonSet) (attributes, error)
}

type persistentVolumeClaimAction interface {
	ComputeAttributes(corev1.PersistentVolumeClaim) (attributes, error)
}
type persistentVolumeAction interface {
	ComputeAttributes(corev1.PersistentVolume) (attributes, error)
}
type ingressAction interface {
	ComputeAttributes(netv1.Ingress) (attributes, error)
}
type endpointsAction interface {
	ComputeAttributes(corev1.Endpoints) (attributes, error)
}

func (proc *K8sEventsProcessor) RunActions(obj metav1.Object) (attributes, error) {
	switch typed := obj.(type) {
	case *corev1.Pod:
		return proc.runPodActions(*typed)
	case *corev1.Node:
		return proc.runNodeActions(*typed)
	case *corev1.Service:
		return proc.runServiceActions(*typed)
	case *batchv1.Job:
		return proc.runJobActions(*typed)
	case *batchv1.CronJob:
		return proc.runCronJobActions(*typed)
	case *appsv1.DaemonSet:
		return proc.runDaemonSetActions(*typed)
	case *appsv1.StatefulSet:
		return proc.runStatefulSetActions(*typed)
	case *corev1.PersistentVolume:
		return proc.runPersistentVolumeActions(*typed)
	case *corev1.PersistentVolumeClaim:
		return proc.runPersistentVolumeClaimActions(*typed)
	case *netv1.Ingress:
		return proc.runIngressActions(*typed)
	case *corev1.ServiceAccount:
		return proc.runServiceAccountActions(*typed)
	case *corev1.Endpoints:
		return proc.runEndpointsActions(*typed)
	}
	return attributes{}, nil
}

func (m *K8sEventsProcessor) runPodActions(pod corev1.Pod) (attributes, error) {
	res := attributes{}
	for _, action := range m.podActions {
		atts, err := action.ComputeAttributes(pod)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runNodeActions(node corev1.Node) (attributes, error) {
	res := attributes{}
	for _, action := range m.nodeActions {
		atts, err := action.ComputeAttributes(node)
		if err != nil {
			return res, err
		}
		// we can do this without worrying about overriding facets since we
		// design facets whithin an entity to have different keys
		for k, v := range atts {
			res[k] = v
		}
	}
	return res, nil
}

func (m *K8sEventsProcessor) runJobActions(job batchv1.Job) (attributes, error) {
	res := attributes{}
	for _, action := range m.jobActions {
		atts, err := action.ComputeAttributes(job)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runCronJobActions(cronJob batchv1.CronJob) (attributes, error) {
	res := attributes{}
	for _, action := range m.cronJobActions {
		atts, err := action.ComputeAttributes(cronJob)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runStatefulSetActions(statefulset appsv1.StatefulSet) (attributes, error) {
	res := attributes{}
	for _, action := range m.statefulSetActions {
		atts, err := action.ComputeAttributes(statefulset)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runDaemonSetActions(daemonset appsv1.DaemonSet) (attributes, error) {
	res := attributes{}
	for _, action := range m.daemonSetActions {
		atts, err := action.ComputeAttributes(daemonset)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runPersistentVolumeActions(pvc corev1.PersistentVolume) (attributes, error) {
	res := attributes{}
	for _, action := range m.persistentVolumeActions {
		atts, err := action.ComputeAttributes(pvc)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runPersistentVolumeClaimActions(pvc corev1.PersistentVolumeClaim) (attributes, error) {
	res := attributes{}
	for _, action := range m.persistentVolumeClaimActions {
		atts, err := action.ComputeAttributes(pvc)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runIngressActions(ingress netv1.Ingress) (attributes, error) {
	res := attributes{}
	for _, action := range m.ingressActions {
		atts, err := action.ComputeAttributes(ingress)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runServiceAccountActions(serviceAccount corev1.ServiceAccount) (attributes, error) {
	res := attributes{}
	for _, action := range m.serviceAccountActions {
		atts, err := action.ComputeAttributes(serviceAccount)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runEndpointsActions(endpoints corev1.Endpoints) (attributes, error) {
	res := attributes{}
	for _, action := range m.endpointsActions {
		atts, err := action.ComputeAttributes(endpoints)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}

func (m *K8sEventsProcessor) runServiceActions(service corev1.Service) (attributes, error) {
	res := attributes{}
	for _, action := range m.serviceActions {
		atts, err := action.ComputeAttributes(service)
		if err != nil {
			return res, err
		}
		res.addAttributes(atts)
	}
	return res, nil
}
