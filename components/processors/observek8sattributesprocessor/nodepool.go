package observek8sattributesprocessor

const (
	NodePoolAttributeKey = "nodePool"
)

type NodePoolAction struct{}

func NewNodePoolAction() NodePoolAction {
	return NodePoolAction{}
}

var providerNodepoolLabels = map[string]struct{}{
	"eks.amazonaws.com/nodegroup":        {}, // AWS
	"cloud.google.com/gke-nodepool":      {}, // GCP
	"kubernetes.azure.com/agentpool":     {}, // "AKS"
	"doks.digitalocean.com/node-pool-id": {}, // "DOKS"
}

// Discover the Node "pool" facet. This faceit is not provided natively by
// Kubernetes, so it will be present only when using a managed
// deployment/service provided by either of the vendors listed above.
func (NodePoolAction) ComputeAttributes(obj any) (attributes, error) {
	node, err := getNode(obj)
	if err != nil {
		return nil, err
	}

	pool := "none"
	for label, value := range node.Labels {
		if _, found := providerNodepoolLabels[label]; found {
			pool = value
			break
		}
	}
	return attributes{NodePoolAttributeKey: pool}, nil
}
