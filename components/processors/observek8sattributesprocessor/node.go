package observek8sattributesprocessor

func filterNodeEvents(event K8sEvent) bool {
	return event.Kind == "Node"
}
