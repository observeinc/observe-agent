package processdiscoveryreceiver

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const defaultKubernetesRefreshInterval = 30 * time.Second

var (
	containerIDPattern = regexp.MustCompile(`(?i)(?:^|[-/:])([a-f0-9]{64})(?:\.scope)?(?:$|/)`)
	podUIDPattern      = regexp.MustCompile(`(?i)pod([a-f0-9][a-f0-9_-]{30,})`)
)

type KubernetesConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	NodeName        string        `mapstructure:"node_name"`
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	KubeconfigPath  string        `mapstructure:"kubeconfig_path"`
}

func (cfg KubernetesConfig) Validate() error {
	if cfg.Enabled && cfg.NodeName == "" {
		return fmt.Errorf("kubernetes.node_name is required when Kubernetes enrichment is enabled")
	}
	if cfg.RefreshInterval < 0 {
		return fmt.Errorf("kubernetes.refresh_interval must not be negative")
	}
	return nil
}

type processEnricher interface {
	Refresh(context.Context)
	Enrich(*ProcessSnapshot)
}

type podAssociation struct {
	containerID        string
	containerName      string
	containerRuntime   string
	containerImageName string
	containerImageID   string
	podUID             string
	podName            string
	namespace          string
	nodeName           string
	workloadKind       string
	workloadName       string
	workloadUID        string
}

type kubernetesEnricher struct {
	config      KubernetesConfig
	client      kubernetes.Interface
	mu          sync.RWMutex
	byContainer map[string]podAssociation
	byPodUID    map[string]podAssociation
	lastRefresh time.Time
}

func newEmptyEnricher(cfg KubernetesConfig) *kubernetesEnricher {
	return &kubernetesEnricher{
		config:      cfg,
		byContainer: make(map[string]podAssociation),
		byPodUID:    make(map[string]podAssociation),
	}
}

func newKubernetesEnricher(cfg KubernetesConfig) processEnricher {
	if !cfg.Enabled {
		return nil
	}
	if cfg.RefreshInterval == 0 {
		cfg.RefreshInterval = defaultKubernetesRefreshInterval
	}
	clientConfig, err := kubernetesClientConfig(cfg)
	if err != nil {
		return newEmptyEnricher(cfg)
	}
	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return newEmptyEnricher(cfg)
	}
	enricher := newEmptyEnricher(cfg)
	enricher.client = client
	return enricher
}

func kubernetesClientConfig(cfg KubernetesConfig) (*rest.Config, error) {
	if cfg.KubeconfigPath != "" {
		return clientcmd.BuildConfigFromFlags("", cfg.KubeconfigPath)
	}
	return rest.InClusterConfig()
}

func (e *kubernetesEnricher) Refresh(ctx context.Context) {
	if e.client == nil || time.Since(e.lastRefresh) < e.config.RefreshInterval {
		return
	}
	pods, err := e.client.CoreV1().Pods("").List(ctx, metav1.ListOptions{FieldSelector: "spec.nodeName=" + e.config.NodeName})
	if err != nil {
		return
	}
	replicaSets, _ := e.client.AppsV1().ReplicaSets("").List(ctx, metav1.ListOptions{})
	replicaSetOwners := make(map[string]metav1.OwnerReference, len(replicaSets.Items))
	for _, replicaSet := range replicaSets.Items {
		if owner, ok := controllerOwner(replicaSet.OwnerReferences); ok {
			replicaSetOwners[string(replicaSet.UID)] = owner
		}
	}
	associations := make(map[string]podAssociation)
	podAssociations := make(map[string]podAssociation)
	for _, pod := range pods.Items {
		workload, _ := controllerOwner(pod.OwnerReferences)
		if workload.Kind == "ReplicaSet" {
			if owner, ok := replicaSetOwners[string(workload.UID)]; ok {
				workload = owner
			}
		}
		podAssociations[normalizePodUID(string(pod.UID))] = podAssociation{
			podUID: string(pod.UID), podName: pod.Name, namespace: pod.Namespace,
			nodeName: pod.Spec.NodeName, workloadKind: workload.Kind,
			workloadName: workload.Name, workloadUID: string(workload.UID),
		}
		statuses := append([]corev1.ContainerStatus{}, pod.Status.InitContainerStatuses...)
		statuses = append(statuses, pod.Status.ContainerStatuses...)
		statuses = append(statuses, pod.Status.EphemeralContainerStatuses...)
		for _, status := range statuses {
			containerID := normalizeContainerID(status.ContainerID)
			if containerID == "" {
				continue
			}
			associations[containerID] = podAssociation{
				containerID: containerID, containerName: status.Name,
				containerRuntime: containerRuntime(status.ContainerID), containerImageName: status.Image,
				containerImageID: normalizeImageID(status.ImageID), podUID: string(pod.UID),
				podName: pod.Name, namespace: pod.Namespace, nodeName: pod.Spec.NodeName,
				workloadKind: workload.Kind, workloadName: workload.Name, workloadUID: string(workload.UID),
			}
		}
	}
	e.mu.Lock()
	e.byContainer = associations
	e.byPodUID = podAssociations
	e.lastRefresh = time.Now()
	e.mu.Unlock()
}

func (e *kubernetesEnricher) Enrich(snapshot *ProcessSnapshot) {
	containerCandidates := snapshot.ContainerIDCandidates
	if len(containerCandidates) == 0 && snapshot.ContainerID != "" {
		containerCandidates = []string{snapshot.ContainerID}
	}
	podCandidates := snapshot.PodUIDCandidates
	if len(podCandidates) == 0 && snapshot.K8sPodUID != "" {
		podCandidates = []string{normalizePodUID(snapshot.K8sPodUID)}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, candidate := range containerCandidates {
		if association, ok := e.byContainer[candidate]; ok {
			applyContainerAssociation(snapshot, association)
			return
		}
	}

	for _, candidate := range podCandidates {
		if association, ok := e.byPodUID[candidate]; ok {
			applyPodAssociation(snapshot, association)
			snapshot.K8sCorrelationStatus = "resolved"
			return
		}
	}

	if len(containerCandidates) > 0 {
		snapshot.ContainerID = containerCandidates[0]
	}
	snapshot.K8sPodUID = ""
	snapshot.K8sCorrelationStatus = "unresolved"
}

func applyContainerAssociation(snapshot *ProcessSnapshot, association podAssociation) {
	snapshot.ContainerID = association.containerID
	snapshot.ContainerName = association.containerName
	snapshot.ContainerRuntime = association.containerRuntime
	snapshot.ContainerImageName = association.containerImageName
	snapshot.ContainerImageID = association.containerImageID
	applyPodAssociation(snapshot, association)
	snapshot.K8sContainerName = association.containerName
	snapshot.K8sCorrelationStatus = "resolved"
}

func applyPodAssociation(snapshot *ProcessSnapshot, association podAssociation) {
	snapshot.K8sPodUID = association.podUID
	snapshot.K8sPodName = association.podName
	snapshot.K8sNamespaceName = association.namespace
	snapshot.K8sNodeName = association.nodeName
	snapshot.K8sWorkloadKind = association.workloadKind
	snapshot.K8sWorkloadName = association.workloadName
	snapshot.K8sWorkloadUID = association.workloadUID
}

func controllerOwner(owners []metav1.OwnerReference) (metav1.OwnerReference, bool) {
	for _, owner := range owners {
		if owner.Controller != nil && *owner.Controller {
			return owner, true
		}
	}
	return metav1.OwnerReference{}, false
}

func containerRuntime(value string) string {
	if index := strings.Index(value, "://"); index >= 0 {
		return value[:index]
	}
	return ""
}

func normalizeImageID(value string) string {
	if index := strings.Index(value, "://"); index >= 0 {
		return value[index+3:]
	}
	return value
}

func parseCgroupIdentity(data []byte) (string, string) {
	candidates := containerCandidatesFromCgroup(data)
	if len(candidates) == 0 {
		return "", ""
	}
	return candidates[0], innermostPodUID(data)
}

func parseCgroupIdentityForPID(data []byte, _ int32) (string, string) {
	candidates := containerCandidatesFromCgroup(data)
	if len(candidates) == 0 {
		return "", ""
	}
	return candidates[0], innermostPodUID(data)
}

func containerCandidatesFromCgroup(data []byte) []string {
	seen := make(map[string]struct{})
	var ordered []string
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		for _, match := range containerIDPattern.FindAllStringSubmatch(parts[2], -1) {
			id := strings.ToLower(match[1])
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			ordered = append(ordered, id)
		}
	}
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}
	return ordered
}

func podUIDCandidatesFromCgroup(data []byte) []string {
	seen := make(map[string]struct{})
	var ordered []string
	for _, match := range podUIDPattern.FindAllStringSubmatch(string(data), -1) {
		id := normalizePodUID(match[1])
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ordered = append(ordered, id)
	}
	for i, j := 0, len(ordered)-1; i < j; i, j = i+1, j-1 {
		ordered[i], ordered[j] = ordered[j], ordered[i]
	}
	return ordered
}

func innermostPodUID(data []byte) string {
	candidates := podUIDCandidatesFromCgroup(data)
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0]
}

func normalizeContainerID(value string) string {
	if index := strings.Index(value, "://"); index >= 0 {
		value = value[index+3:]
	}
	value = strings.TrimSuffix(value, ".scope")
	if len(value) != 64 {
		return ""
	}
	return strings.ToLower(value)
}

func normalizePodUID(value string) string {
	return strings.ToLower(strings.ReplaceAll(value, "_", "-"))
}
