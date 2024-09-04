package observek8sattributesprocessor

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
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
	ComputeAttributes(v1.Pod) (attributes, error)
}
type nodeAction interface {
	ComputeAttributes(v1.Node) (attributes, error)
}
type jobAction interface {
	ComputeAttributes(batchv1.Job) (attributes, error)
}
type cronJobAction interface {
	ComputeAttributes(batchv1.CronJob) (attributes, error)
}
type daemonSetAction interface {
	ComputeAttributes(appsv1.DaemonSet) (attributes, error)
}

func (proc *K8sEventsProcessor) RunActions(obj metav1.Object) (attributes, error) {
	switch typed := obj.(type) {
	case *v1.Pod:
		return proc.runPodActions(*typed)
	case *v1.Node:
		return proc.runNodeActions(*typed)
	case *batchv1.Job:
		return proc.runJobActions(*typed)
	case *batchv1.CronJob:
		return proc.runCronJobActions(*typed)
	case *appsv1.DaemonSet:
		return proc.runDaemonSetActions(*typed)
	}
	return attributes{}, nil
}

func (m *K8sEventsProcessor) runPodActions(pod v1.Pod) (attributes, error) {
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

func (m *K8sEventsProcessor) runNodeActions(node v1.Node) (attributes, error) {
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
