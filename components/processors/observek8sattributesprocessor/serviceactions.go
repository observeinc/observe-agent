package observek8sattributesprocessor

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ServiceLBIngressAttributeKey   = "loadBalancerIngress"
	ServiceExternalIPsAttributeKey = "externalIPs"
	ServicePortsAttributeKey       = "ports"
)

// loadBalancerStatusStringer behaves mostly like a string interface and converts the given status to a string.
func loadBalancerStatusStringer(s corev1.LoadBalancerStatus) []string {
	ingress := s.Ingress
	result := sets.NewString()
	for i := range ingress {
		if ingress[i].IP != "" {
			result.Insert(ingress[i].IP)
		} else if ingress[i].Hostname != "" {
			result.Insert(ingress[i].Hostname)
		}
	}

	return result.List()
}

// Generates the service's externalIPs, based on the service type
// Returns an array of externalIPs and TRUE if there is any external IP.
// Returns an array with a single string "None", "Pending" or "Unknown" and FALSE if there are no external IPs.
func getServiceExternalIPs(svc corev1.Service) ([]string, bool) {
	switch svc.Spec.Type {
	case corev1.ServiceTypeClusterIP:
		if len(svc.Spec.ExternalIPs) > 0 {
			return svc.Spec.ExternalIPs, true
		}
		return []string{"None"}, false
	case corev1.ServiceTypeNodePort:
		if len(svc.Spec.ExternalIPs) > 0 {
			return svc.Spec.ExternalIPs, true
		}
		return []string{"None"}, false
	case corev1.ServiceTypeLoadBalancer:
		results := loadBalancerStatusStringer(svc.Status.LoadBalancer)
		results = append(results, svc.Spec.ExternalIPs...)
		if len(results) > 0 {
			return results, true
		}
		return []string{"Pending"}, false
	case corev1.ServiceTypeExternalName:
		return []string{svc.Spec.ExternalName}, true
	}
	return []string{"Unknown"}, false
}

type ServiceExternalIPsAction struct{}

func NewServiceExternalIPsAction() ServiceExternalIPsAction {
	return ServiceExternalIPsAction{}
}

// Generates the Service "loadBalancerIngress" facet.
// We return an array of IPs (even if there's only a single IP) if there is at least one.
// We set the facet to a string (no array in this case) as follows:
//
// - "None" --> service != LoadBalancer and there are no external IPs
// - "Pending" --> service is a LoadBalancer and there are no external IPs
// - "Unknown" --> service is of an unknown type
func (ServiceExternalIPsAction) ComputeAttributes(service corev1.Service) (attributes, error) {
	if externalIPs, ok := getServiceExternalIPs(service); ok {
		return attributes{ServiceExternalIPsAttributeKey: externalIPs}, nil
	} else {
		return attributes{ServiceExternalIPsAttributeKey: externalIPs[0]}, nil
	}
}

// ---------------------------------- Service "loadBalancerIngress" ----------------------------------

type ServiceLBIngressAction struct{}

func NewServiceLBIngressAction() ServiceLBIngressAction {
	return ServiceLBIngressAction{}
}

// Generates the Service "loadBalancerIngress" facet.
func (ServiceLBIngressAction) ComputeAttributes(service corev1.Service) (attributes, error) {
	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return attributes{}, nil
	}

	ingress := loadBalancerStatusStringer(service.Status.LoadBalancer)
	return attributes{ServiceLBIngressAttributeKey: strings.Join(ingress, ",")}, nil

}

// ---------------------------------- Service "selector" ----------------------------------

type ServiceSelectorAction struct{}

func NewServiceSelectorAction() ServiceSelectorAction {
	return ServiceSelectorAction{}
}

// Generates the Service "selector" facet.
func (ServiceSelectorAction) ComputeAttributes(service corev1.Service) (attributes, error) {
	selectorString := FormatLabels(service.Spec.Selector)
	return attributes{DaemonSetSelectorAttributeKey: selectorString}, nil

}

// ---------------------------------- Service "ports" ----------------------------------

type ServicePortsAction struct{}

func NewServicePortsAction() ServicePortsAction {
	return ServicePortsAction{}
}

func makePortString(ports []corev1.ServicePort) string {
	pieces := make([]string, len(ports))
	for ix := range ports {
		port := &ports[ix]
		pieces[ix] = fmt.Sprintf("%d/%s", port.Port, port.Protocol)
		if port.NodePort > 0 {
			pieces[ix] = fmt.Sprintf("%d:%d/%s", port.Port, port.NodePort, port.Protocol)
		}
	}
	return strings.Join(pieces, ",")
}

// Generates the Service "ports" facet.
func (ServicePortsAction) ComputeAttributes(service corev1.Service) (attributes, error) {
	return attributes{ServicePortsAttributeKey: makePortString(service.Spec.Ports)}, nil

}
