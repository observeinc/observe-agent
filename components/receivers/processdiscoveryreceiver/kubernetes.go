package processdiscoveryreceiver

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
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
	containerIDPattern   = regexp.MustCompile(`(?i)(?:^|[-/:])([a-f0-9]{64})(?:\.scope)?(?:$|/)`)
	containerLeafPattern = regexp.MustCompile(`(?i)^(?:(?:cri-containerd|docker|crio)-)?([a-f0-9]{64})$`)
	podUIDPattern        = regexp.MustCompile(`(?i)pod([a-f0-9][a-f0-9_-]{30,})`)
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
	lastRefresh time.Time
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
		return &kubernetesEnricher{config: cfg, byContainer: make(map[string]podAssociation)}
	}
	client, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		return &kubernetesEnricher{config: cfg, byContainer: make(map[string]podAssociation)}
	}
	return &kubernetesEnricher{config: cfg, client: client, byContainer: make(map[string]podAssociation)}
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
	for _, pod := range pods.Items {
		workload, _ := controllerOwner(pod.OwnerReferences)
		if workload.Kind == "ReplicaSet" {
			if owner, ok := replicaSetOwners[string(workload.UID)]; ok {
				workload = owner
			}
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
	e.lastRefresh = time.Now()
	e.mu.Unlock()
}

func (e *kubernetesEnricher) Enrich(snapshot *ProcessSnapshot) {
	if snapshot.ContainerID == "" {
		snapshot.K8sCorrelationStatus = "unresolved"
		return
	}
	e.mu.RLock()
	association, ok := e.byContainer[snapshot.ContainerID]
	e.mu.RUnlock()
	if !ok {
		// A pod UID parsed from a cgroup path is not authoritative without a
		// matching node-local pod/container association. This also prevents an
		// outer host container from being presented as a minikube workload.
		snapshot.K8sPodUID = ""
		snapshot.K8sCorrelationStatus = "unresolved"
		return
	}
	if snapshot.K8sPodUID != "" && normalizePodUID(snapshot.K8sPodUID) != normalizePodUID(association.podUID) {
		snapshot.K8sCorrelationStatus = "ambiguous"
		return
	}
	snapshot.ContainerID = association.containerID
	snapshot.ContainerName = association.containerName
	snapshot.ContainerRuntime = association.containerRuntime
	snapshot.ContainerImageName = association.containerImageName
	snapshot.ContainerImageID = association.containerImageID
	snapshot.K8sPodUID = association.podUID
	snapshot.K8sPodName = association.podName
	snapshot.K8sNamespaceName = association.namespace
	snapshot.K8sContainerName = association.containerName
	snapshot.K8sNodeName = association.nodeName
	snapshot.K8sWorkloadKind = association.workloadKind
	snapshot.K8sWorkloadName = association.workloadName
	snapshot.K8sWorkloadUID = association.workloadUID
	snapshot.K8sCorrelationStatus = "resolved"
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
	text := string(data)
	containerMatch := containerIDPattern.FindStringSubmatch(text)
	podMatch := podUIDPattern.FindStringSubmatch(text)
	containerID, podUID := "", ""
	if len(containerMatch) > 1 {
		containerID = strings.ToLower(containerMatch[1])
	}
	if len(podMatch) > 1 {
		podUID = normalizePodUID(podMatch[1])
	}
	return containerID, podUID
}

func parseCgroupIdentityForPID(data []byte, pid int32) (string, string) {
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, ":", 3)
		if len(parts) != 3 {
			continue
		}
		path := parts[2]
		if !strings.Contains(path, "pod") {
			if match := containerIDPattern.FindStringSubmatch(path); len(match) > 1 {
				return strings.ToLower(match[1]), ""
			}
			continue
		}

		leaf := strings.TrimSuffix(path[strings.LastIndex(path, "/")+1:], ".scope")
		if leaf == strconv.Itoa(int(pid)) {
			continue
		}
		containerMatch := containerLeafPattern.FindStringSubmatch(leaf)
		podMatch := podUIDPattern.FindStringSubmatch(path)
		if len(containerMatch) < 2 || len(podMatch) < 2 {
			continue
		}
		return strings.ToLower(containerMatch[1]), normalizePodUID(podMatch[1])
	}
	return "", ""
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
