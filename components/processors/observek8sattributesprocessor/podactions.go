package observek8sattributesprocessor

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
)

const (
	PodStatusAttributeKey = "status"
	// from https://github.com/kubernetes/kubernetes/blob/abe6321296123aaba8e83978f7d17951ab1b64fd/pkg/util/node/node.go#L43
	nodeUnreachablePodReason = "NodeLost"

	PodContainerRestartsAttributeKey = "restarts"
	PodTotalContainersAttributeKey   = "total_containers"
	PodReadyContainersAttributeKey   = "ready_containers"

	PodReadinessGatesReadyAttributeKey = "readinessGatesReady"
	PodReadinessGatesTotalAttributeKey = "readinessGatesTotal"

	PodConditionsAttributeKey = "conditions"

	PodCronJobNameAttributeKey = "cronJobName"
	OwnerKindCronJob           = "CronJob"

	PodJobNameAttributeKey = "jobName"
	OwnerKindJob           = "Job"

	PodDaemonSetNameAttributeKey = "daemonSetName"
	OwnerKindDaemonSet           = "DaemonSet"

	PodStatefulSetNameAttributeKey = "statefulSetName"
	OwnerKindStatefulSet           = "StatefulSet"
)

// ---------------------------------- Pod "status" ----------------------------------

type PodStatusAction struct{}

func NewPodStatusAction() PodStatusAction {
	return PodStatusAction{}
}

// Generates the Pod "status" facet.
func (PodStatusAction) ComputeAttributes(pod v1.Pod) (attributes, error) {
	// based on https://github.com/kubernetes/kubernetes/blob/0d3b859af81e6a5f869a7766c8d45afd1c600b04/pkg/printers/internalversion/printers.go#L901
	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false
	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}
	if !initializing {
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}

	if pod.DeletionTimestamp != nil && pod.Status.Reason == nodeUnreachablePodReason {
		reason = "Unknown"
	} else if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	return attributes{PodStatusAttributeKey: reason, "test": false}, nil
}

// ---------------------------------- various Pod "counts" ----------------------------------

// This action computes various facets for Pod by aggregating "status" values
// across all containers of a Pod.
//
// We compute more facets into a single action to avoid iterating over the
// same slice multiple times in different actions.
type PodContainersCountsAction struct{}

func NewPodContainersCountsAction() PodContainersCountsAction {
	return PodContainersCountsAction{}
}

func (PodContainersCountsAction) ComputeAttributes(pod v1.Pod) (attributes, error) {
	// we use int32 since containerStatuses.restartCount is int32
	var restartsCount int32
	// We don't need to use a hash set on the container ID for these two facets,
	// since the containerStatuses contain one entry per container.
	var readyContainers int64
	var allContainers int64

	for _, stat := range pod.Status.ContainerStatuses {
		restartsCount += stat.RestartCount
		allContainers++
		if stat.Ready {
			readyContainers++
		}
	}

	// Returning map[string]any will make the processor add its elements as
	// separate facets, rather than adding the whole map under the key of this action
	return attributes{
		PodContainerRestartsAttributeKey: int64(restartsCount),
		PodTotalContainersAttributeKey:   allContainers,
		PodReadyContainersAttributeKey:   readyContainers,
	}, nil
}

// ---------------------------------- Pod "readiness" ----------------------------------

type PodReadinessAction struct{}

func NewPodReadinessAction() PodReadinessAction {
	return PodReadinessAction{}
}

func (PodReadinessAction) ComputeAttributes(pod v1.Pod) (attributes, error) {
	readinessGatesReady := 0

	if len(pod.Spec.ReadinessGates) > 0 {
		for _, readinessGate := range pod.Spec.ReadinessGates {
			conditionType := readinessGate.ConditionType
			for _, condition := range pod.Status.Conditions {
				if condition.Type == conditionType {
					if condition.Status == v1.ConditionTrue {
						readinessGatesReady++
					}
					break
				}
			}
		}
	}

	return attributes{
		PodReadinessGatesTotalAttributeKey: len(pod.Spec.ReadinessGates),
		PodReadinessGatesReadyAttributeKey: readinessGatesReady,
	}, nil
}

// ---------------------------------- Pod "conditions" ----------------------------------

// Gather all Pod conditions into a single facet named "conditions"
type PodConditionsAction struct{}

func NewPodConditionsAction() PodConditionsAction {
	return PodConditionsAction{}
}

func (PodConditionsAction) ComputeAttributes(pod v1.Pod) (attributes, error) {
	conditions := attributes{}
	for _, cond := range pod.Status.Conditions {
		// Assumes that k8s doesn't return multiple conditions of the same type.
		// If that were to happen, we just overwrite the previous ones
		conditions[string(cond.Type)] = string(cond.Status)
	}

	// return a map[string]any with a single value.
	// The key is the name of the face (and of this action),
	// This facet's value is a map itself with k-v pairs, keyed with strings
	return attributes{PodConditionsAttributeKey: conditions}, nil
}

// ---------------------------------- Pod "jobName" ----------------------------------

type PodJobNameAction struct{}

func NewPodJobAction() PodJobNameAction {
	return PodJobNameAction{}
}

// Name of the job this Pod belongs to (only present if the owner is a Job resource)
func (PodJobNameAction) ComputeAttributes(pod v1.Pod) (attributes, error) {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == OwnerKindJob {
			return attributes{PodJobNameAttributeKey: ref.Name}, nil
		}
	}
	return attributes{}, nil
}

// ---------------------------------- Pod "cronJobName" ----------------------------------

type PodCronJobNameAction struct{}

func NewPodCronJobNameAction() PodCronJobNameAction {
	return PodCronJobNameAction{}
}

// Name of the cronJob this Pod belongs to (only present if the owner is a CronJobName resource)
func (PodCronJobNameAction) ComputeAttributes(pod v1.Pod) (attributes, error) {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == OwnerKindCronJob {
			return attributes{PodCronJobNameAttributeKey: ref.Name}, nil
		}
	}
	return attributes{}, nil
}

// ---------------------------------- Pod "daemonSetName" ----------------------------------

type PodDaemonSetNameAction struct{}

func NewPodDaemonSetNameAction() PodDaemonSetNameAction {
	return PodDaemonSetNameAction{}
}

// Name of the cronJob this Pod belongs to (only present if the owner is a DaemonSetName resource)
func (PodDaemonSetNameAction) ComputeAttributes(pod v1.Pod) (attributes, error) {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == OwnerKindDaemonSet {
			return attributes{PodDaemonSetNameAttributeKey: ref.Name}, nil
		}
	}
	return attributes{}, nil
}

// ---------------------------------- Pod "statefulSetName" ----------------------------------

type PodStatefulSetNameAction struct{}

func NewPodStatefulSetNameAction() PodStatefulSetNameAction {
	return PodStatefulSetNameAction{}
}

// Name of the cronJob this Pod belongs to (only present if the owner is a StatefulSetName resource)
func (PodStatefulSetNameAction) ComputeAttributes(pod v1.Pod) (attributes, error) {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == OwnerKindStatefulSet {
			return attributes{PodStatefulSetNameAttributeKey: ref.Name}, nil
		}
	}
	return attributes{}, nil
}
