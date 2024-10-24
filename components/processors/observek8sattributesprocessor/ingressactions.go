package observek8sattributesprocessor

import (
	"strings"

	netv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/sets"
)

const (
	IngressRulesAttributeKey        = "rules"
	IngressLoadBalancerAttributeKey = "loadBalancer"
	hostKey                         = "host"
	rulesKey                        = "rules"
	httpRulesKey                    = "httpRules"
	pathKey                         = "path"
	backendKey                      = "backend"
	serviceKey                      = "service"
	resourceKey                     = "resource"
	portKey                         = "port"
	nameKey                         = "name"
)

// formatIngressRules converts a slice of IngressRules to a minimal JSON representation.
// The json structure is:
//
//		{
//		  "host": "example.com",    (or "*" if no host is specified)
//		  "rules": [
//		    {
//		      "path": "/app1",
//		      "backend": {
//		        "service": {        (inside "backend" could be either "service" or "resource")
//		          "name": "app1-service",
//		          "port": 8080       (could be either a number or a string)
//		        }
//		      }
//		    },
//		    {
//		      "path": "/app2",
//		      "backend": {
//		        "resource": "app2-resource",   (alternative to service)
//	            (no port here, just the name of the resource)
//		      }
//		    }
//		  ]
//		}
func formatIngressRules(rules []netv1.IngressRule) []attributes {
	var ret []attributes

	for _, rule := range rules {
		host := rule.Host
		if host == "" {
			host = "*"
		}

		var httpRules []attributes
		if rule.HTTP != nil {
			for _, path := range rule.HTTP.Paths {

				ruleInfo := attributes{
					pathKey: path.Path,
				}
				backend := path.Backend
				backendAttrs := attributes{}
				if backend.Service != nil {
					service := backend.Service
					serviceAttrs := attributes{
						nameKey: service.Name,
					}
					// Remove one level of indentation and use either the port
					// name or the number as "port" directly inside "service",
					// since we can use values of any type.
					if service.Port.Name != "" {
						serviceAttrs[portKey] = service.Port.Name
					} else {
						serviceAttrs[portKey] = service.Port.Number
					}
					backendAttrs[serviceKey] = serviceAttrs
				} else {
					backendAttrs[resourceKey] = backend.Resource.Name
				}
				ruleInfo[backendKey] = backendAttrs

				httpRules = append(httpRules, ruleInfo)
			}
		}

		ret = append(ret, attributes{
			hostKey:      host,
			httpRulesKey: httpRules,
		})
	}

	return ret
}

// ---------------------------------- Ingress "rules" ----------------------------------

type IngressRulesAction struct{}

func NewIngressRulesAction() IngressRulesAction {
	return IngressRulesAction{}
}

// Generates the Ingress "rules" facet.
func (IngressRulesAction) ComputeAttributes(ingress netv1.Ingress) (attributes, error) {
	rules := formatIngressRules(ingress.Spec.Rules)
	return attributes{IngressRulesAttributeKey: rules}, nil
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
