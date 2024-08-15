package observek8sattributesprocessor

func filterPodEvents(event K8sEvent) bool {
	return event.Kind == "Pod"
}
