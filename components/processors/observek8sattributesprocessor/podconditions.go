package observek8sattributesprocessor

const (
	// This action will be ignored and not written in any of the facets, since
	// we return map[string]any
	PodConditionsAttributeKey = "conditions"
)

// Gather all Pod conditions into a single facet named "conditions"
// (with actual type: map[string]bool)
type PodConditionsAction struct{}

func NewPodConditionsAction() PodConditionsAction {
	return PodConditionsAction{}
}

func (PodConditionsAction) ComputeAttributes(obj any) (attributes, error) {
	pod, err := getPod(obj)
	if err != nil {
		return nil, err
	}
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
