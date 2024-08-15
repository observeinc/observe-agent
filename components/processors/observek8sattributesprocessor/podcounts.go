package observek8sattributesprocessor

import (
	"encoding/json"
	"errors"

	"go.opentelemetry.io/collector/pdata/plog"
	v1 "k8s.io/api/core/v1"
)

const (
	// This action will be ignored and not written in any of the facets, since
	// we return map[string]any
	PodContainerRestartsAttributeKey = "restarts"
	PodTotalContainersAttributeKey   = "total_containers"
	PodReadyContainersAttributeKey   = "ready_containers"
)

// This action computes various facets for Pod by aggregating "status" values
// across all containers of a Pod.
//
// We compute more facets into a single action to avoid iterating over the
// same slice multiple times in different actions.
var PodContainersCountsAction = K8sEventProcessorAction{
	ComputeAttributes: getPodCounts,
	FilterFn:          filterPodEvents,
}

func getPodCounts(objLog plog.LogRecord) (attributes, error) {
	var p v1.Pod
	err := json.Unmarshal([]byte(objLog.Body().AsString()), &p)
	if err != nil {
		return nil, errors.New("Unknown")
	}
	// we use int32 since containerStatuses.restartCount is int32
	var restartsCount int32
	// We don't need to use a hash set on the container ID for these two facets,
	// since the containerStatuses contain one entry per container.
	var readyContainers int64
	var allContainers int64

	for _, stat := range p.Status.ContainerStatuses {
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
