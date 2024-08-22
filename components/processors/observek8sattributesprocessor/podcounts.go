package observek8sattributesprocessor

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
type PodContainersCountsAction struct{}

func NewPodContainersCountsAction() PodContainersCountsAction {
	return PodContainersCountsAction{}
}

func (PodContainersCountsAction) ComputeAttributes(obj any) (attributes, error) {
	pod, err := getPod(obj)
	if err != nil {
		return nil, err
	}
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
