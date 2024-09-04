package observek8sattributesprocessor

import (
	"net"
	"strconv"

	corev1 "k8s.io/api/core/v1"
)

const (
	EnpdointsAttributeKey = "endpoints"
)

// ---------------------------------- Endpoints "endpoints" ----------------------------------

type EndpointsStatusAction struct{}

func NewEndpointsStatusAction() EndpointsStatusAction {
	return EndpointsStatusAction{}
}

// Generates the Endpoints "endpoints" facet, which is a list of all individual endpoints, encoded as strings
func (EndpointsStatusAction) ComputeAttributes(endpoints corev1.Endpoints) (attributes, error) {
	list := []string{}
	for _, ss := range endpoints.Subsets {
		if len(ss.Ports) == 0 {
			// It's possible to have headless services with no ports.
			for i := range ss.Addresses {
				list = append(list, ss.Addresses[i].IP)
			}
			// avoid nesting code too deeply
			continue
		}

		// "Normal" services with ports defined.
		for _, port := range ss.Ports {
			for i := range ss.Addresses {
				addr := &ss.Addresses[i]
				hostPort := net.JoinHostPort(addr.IP, strconv.Itoa(int(port.Port)))
				list = append(list, hostPort)
			}
		}
	}
	return attributes{EnpdointsAttributeKey: list}, nil
}
