package observek8sattributesprocessor

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	ServiceLBIngressAttributeKey = "loadBalancerIngress"
	ServicePortsAttributeKey     = "ports"
)

// ---------------------------------- Service "loadBalancerIngress" ----------------------------------

type ServiceLBIngressAction struct{}

func NewServiceLBIngressAction() ServiceLBIngressAction {
	return ServiceLBIngressAction{}
}

// loadBalancerStatusStringer behaves mostly like a string interface and converts the given status to a string.
// Adapted from https://github.com/kubernetes/kubernetes/blob/0d3b859af81e6a5f869a7766c8d45afd1c600b04/pkg/printers/internalversion/printers.go#L1275
func loadBalancerStatusStringer(s corev1.LoadBalancerStatus) string {
	ingress := s.Ingress
	result := sets.NewString()
	for i := range ingress {
		if ingress[i].IP != "" {
			result.Insert(ingress[i].IP)
		} else if ingress[i].Hostname != "" {
			result.Insert(ingress[i].Hostname)
		}
	}

	r := strings.Join(result.List(), ",")
	return r
}

// Adapted from https://github.com/kubernetes/kubernetes/blob/0d3b859af81e6a5f869a7766c8d45afd1c600b04/pkg/printers/internalversion/printers.go#L1293
func getServiceExternalIP(svc *corev1.Service) string {
	lbIps := loadBalancerStatusStringer(svc.Status.LoadBalancer)
	if len(svc.Spec.ExternalIPs) > 0 {
		results := []string{}
		if len(lbIps) > 0 {
			results = append(results, strings.Split(lbIps, ",")...)
		}
		results = append(results, svc.Spec.ExternalIPs...)
		return strings.Join(results, ",")
	}
	if len(lbIps) > 0 {
		return lbIps
	}
	return "<pending>"
}

// Generates the Service "loadBalancerIngress" facet.
func (ServiceLBIngressAction) ComputeAttributes(service corev1.Service) (attributes, error) {
	if service.Spec.Type != corev1.ServiceTypeLoadBalancer {
		return attributes{}, nil
	}

	return attributes{ServiceLBIngressAttributeKey: getServiceExternalIP(&service)}, nil

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
