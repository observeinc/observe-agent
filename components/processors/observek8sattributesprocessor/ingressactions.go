package observek8sattributesprocessor

import (
	"strings"

	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	IngressRulesAttributeKey        = "rules"
	IngressLoadBalancerAttributeKey = "loadBalancer"
)

// Adapted from https://github.com/kubernetes/kubernetes/blob/0d3b859af81e6a5f869a7766c8d45afd1c600b04/pkg/printers/internalversion/printers.go#L1373
func formatIngressRules(rules []netv1.IngressRule) string {
	list := []string{}
	for _, rule := range rules {
		list = append(list, rule.Host)
	}
	if len(list) == 0 {
		return "*"
	}
	ret := strings.Join(list, ",")
	return ret
}

// ---------------------------------- Ingress "rules" ----------------------------------
type IngressRulesAction struct{}

func NewIngressRulesAction() IngressRulesAction {
	return IngressRulesAction{}
}

// Generates the Ingress "rules" facet.
func (IngressRulesAction) ComputeAttributes(ingress netv1.Ingress) (attributes, error) {
	return attributes{IngressRulesAttributeKey: formatIngressRules(ingress.Spec.Rules)}, nil
}

// ---------------------------------- Ingress "loadBalancer" ----------------------------------
type IngressLoadBalancerAction struct{}

func NewIngressLoadBalancerAction() IngressLoadBalancerAction {
	return IngressLoadBalancerAction{}
}

// Adapted from https://github.com/kubernetes/kubernetes/blob/0d3b859af81e6a5f869a7766c8d45afd1c600b04/pkg/printers/internalversion/printers.go#L1420
// (removed wide option and always extract full info)
func ingressLoadBalancerStatusStringer(s netv1.IngressLoadBalancerStatus) string {
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

// Generates the Ingress "loadBalancer" facet.
func (IngressLoadBalancerAction) ComputeAttributes(ingress netv1.Ingress) (attributes, error) {
	return attributes{IngressLoadBalancerAttributeKey: ingressLoadBalancerStatusStringer(ingress.Status.LoadBalancer)}, nil
}
