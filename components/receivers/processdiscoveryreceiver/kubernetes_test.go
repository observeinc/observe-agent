package processdiscoveryreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testContainerID = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestParseCgroupIdentity(t *testing.T) {
	containerID, podUID := parseCgroupIdentityForPID([]byte(
		"0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-pod4b1f4f77_49e6_4e30_98db_e98055c8a54f.slice/cri-containerd-"+testContainerID+".scope\n",
	), 123)
	assert.Equal(t, testContainerID, containerID)
	assert.Equal(t, "4b1f4f77-49e6-4e30-98db-e98055c8a54f", podUID)
}

func TestParseCgroupIdentityRejectsShortContainerID(t *testing.T) {
	containerID, _ := parseCgroupIdentity([]byte("0::/kubepods/podabc/0123456789ab"))
	assert.Empty(t, containerID)
}

func TestParseCgroupIdentityRejectsInheritedHostPodCgroup(t *testing.T) {
	data := []byte("0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-poda7b74024_f357_4d7e_858f_f0da16d1be6f.slice/123.scope\n")
	containerID, podUID := parseCgroupIdentityForPID(data, 123)
	assert.Empty(t, containerID)
	assert.Empty(t, podUID)
}

func TestParseCgroupIdentityKeepsContainerLeaf(t *testing.T) {
	data := []byte("0::/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-poda7b74024_f357_4d7e_858f_f0da16d1be6f.slice/docker-" + testContainerID + ".scope\n")
	containerID, podUID := parseCgroupIdentityForPID(data, 123)
	assert.Equal(t, testContainerID, containerID)
	assert.Equal(t, "a7b74024-f357-4d7e-858f-f0da16d1be6f", podUID)
}

func TestParseCgroupIdentityForPIDSupportsContainerLayouts(t *testing.T) {
	testCases := []struct {
		name string
		path string
	}{
		{name: "docker", path: "/docker/" + testContainerID},
		{name: "systemd docker", path: "/system.slice/docker-" + testContainerID + ".scope"},
		{name: "containerd", path: "/system.slice/cri-containerd-" + testContainerID + ".scope"},
		{name: "crio", path: "/system.slice/crio-" + testContainerID + ".scope"},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			containerID, podUID := parseCgroupIdentityForPID([]byte("0::"+testCase.path+"\n"), 123)
			assert.Equal(t, testContainerID, containerID)
			assert.Empty(t, podUID)
		})
	}
}

func TestParseCgroupIdentityForPIDSupportsCRIOPod(t *testing.T) {
	data := []byte("0::/kubepods.slice/kubepods-burstable-pod4b1f4f77_49e6_4e30_98db_e98055c8a54f.slice/crio-" + testContainerID + ".scope\n")
	containerID, podUID := parseCgroupIdentityForPID(data, 123)
	assert.Equal(t, testContainerID, containerID)
	assert.Equal(t, "4b1f4f77-49e6-4e30-98db-e98055c8a54f", podUID)
}

func TestKubernetesEnricherResolvesContainer(t *testing.T) {
	enricher := &kubernetesEnricher{
		byContainer: map[string]podAssociation{
			testContainerID: {
				containerID: testContainerID, containerName: "app",
				podUID: "4b1f4f77-49e6-4e30-98db-e98055c8a54f", podName: "api", namespace: "prod",
			},
		},
	}
	snapshot := ProcessSnapshot{WorkloadContext: WorkloadContext{ContainerID: testContainerID, K8sPodUID: "4b1f4f77_49e6_4e30_98db_e98055c8a54f"}}
	enricher.Enrich(&snapshot)
	require.Equal(t, "resolved", snapshot.K8sCorrelationStatus)
	assert.Equal(t, "prod", snapshot.K8sNamespaceName)
	assert.Equal(t, "api", snapshot.K8sPodName)
	assert.Equal(t, "app", snapshot.K8sContainerName)
}

func TestKubernetesEnricherContainerMatchIsAuthoritative(t *testing.T) {
	enricher := &kubernetesEnricher{byContainer: map[string]podAssociation{
		testContainerID: {
			containerID: testContainerID, containerName: "app", containerRuntime: "containerd",
			containerImageName: "example/app:v1", containerImageID: "sha256:abc",
			podUID: "expected", podName: "api", namespace: "prod", nodeName: "node-a",
			workloadKind: "Deployment", workloadName: "api", workloadUID: "workload-uid",
		},
	}}
	snapshot := ProcessSnapshot{WorkloadContext: WorkloadContext{ContainerID: testContainerID, K8sPodUID: "different"}}
	enricher.Enrich(&snapshot)
	assert.Equal(t, "resolved", snapshot.K8sCorrelationStatus)
	assert.Equal(t, "expected", snapshot.K8sPodUID)
	assert.Equal(t, "api", snapshot.K8sPodName)
	assert.Equal(t, "containerd", snapshot.ContainerRuntime)
	assert.Equal(t, "example/app:v1", snapshot.ContainerImageName)
	assert.Equal(t, "Deployment", snapshot.K8sWorkloadKind)
	assert.Equal(t, "workload-uid", snapshot.K8sWorkloadUID)
}

func TestKubernetesEnricherResolvesInnermostNestedContainer(t *testing.T) {
	const innerContainerID = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	enricher := &kubernetesEnricher{byContainer: map[string]podAssociation{
		innerContainerID: {
			containerID: innerContainerID, containerName: "workload",
			podUID: "inner-uid", podName: "inner-pod", namespace: "apps",
		},
	}}
	snapshot := ProcessSnapshot{WorkloadContext: WorkloadContext{
		ContainerID:           innerContainerID,
		ContainerIDCandidates: []string{innerContainerID, testContainerID},
	}}
	enricher.Enrich(&snapshot)
	assert.Equal(t, "resolved", snapshot.K8sCorrelationStatus)
	assert.Equal(t, "apps", snapshot.K8sNamespaceName)
	assert.Equal(t, "inner-pod", snapshot.K8sPodName)
	assert.Equal(t, "workload", snapshot.K8sContainerName)
}

func TestKubernetesEnricherUnresolvedClearsPodIdentity(t *testing.T) {
	enricher := &kubernetesEnricher{
		byContainer: map[string]podAssociation{},
		byPodUID:    map[string]podAssociation{},
	}
	snapshot := ProcessSnapshot{WorkloadContext: WorkloadContext{
		ContainerID:           testContainerID,
		ContainerIDCandidates: []string{testContainerID},
		K8sPodUID:             "cgroup-only-uid",
	}}
	enricher.Enrich(&snapshot)
	assert.Equal(t, "unresolved", snapshot.K8sCorrelationStatus)
	assert.Equal(t, testContainerID, snapshot.ContainerID)
	assert.Empty(t, snapshot.K8sPodUID)
	assert.Empty(t, snapshot.K8sPodName)
}

func TestKubernetesEnricherFallsBackToPodUID(t *testing.T) {
	const pauseContainerID = "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	enricher := &kubernetesEnricher{
		byContainer: map[string]podAssociation{},
		byPodUID: map[string]podAssociation{
			"e08bc522-213a-4ed5-85c4-cefbc7623b71": {
				podUID: "e08bc522-213a-4ed5-85c4-cefbc7623b71", podName: "redis-ephemeral-0",
				namespace: "sandbox", nodeName: "node-a", workloadKind: "StatefulSet",
				workloadName: "redis-ephemeral", workloadUID: "workload-uid",
			},
		},
	}
	snapshot := ProcessSnapshot{WorkloadContext: WorkloadContext{
		ContainerID:           pauseContainerID,
		ContainerIDCandidates: []string{pauseContainerID},
		PodUIDCandidates: []string{
			"e08bc522-213a-4ed5-85c4-cefbc7623b71",
			"a7b74024-f357-4d7e-858f-f0da16d1be6f",
		},
	}}
	enricher.Enrich(&snapshot)
	assert.Equal(t, "resolved", snapshot.K8sCorrelationStatus)
	assert.Equal(t, "sandbox", snapshot.K8sNamespaceName)
	assert.Equal(t, "redis-ephemeral-0", snapshot.K8sPodName)
	assert.Equal(t, "node-a", snapshot.K8sNodeName)
	assert.Equal(t, "StatefulSet", snapshot.K8sWorkloadKind)
	assert.Equal(t, "redis-ephemeral", snapshot.K8sWorkloadName)
	assert.Empty(t, snapshot.K8sContainerName)
	assert.Empty(t, snapshot.ContainerName)
}

func TestKubernetesEnricherPrefersContainerOverPodUID(t *testing.T) {
	enricher := &kubernetesEnricher{
		byContainer: map[string]podAssociation{
			testContainerID: {
				containerID: testContainerID, containerName: "app",
				podUID: "pod-uid", podName: "api", namespace: "prod",
			},
		},
		byPodUID: map[string]podAssociation{
			"pod-uid": {podUID: "pod-uid", podName: "api", namespace: "wrong-if-used"},
		},
	}
	snapshot := ProcessSnapshot{WorkloadContext: WorkloadContext{
		ContainerIDCandidates: []string{testContainerID},
		PodUIDCandidates:      []string{"pod-uid"},
	}}
	enricher.Enrich(&snapshot)
	assert.Equal(t, "resolved", snapshot.K8sCorrelationStatus)
	assert.Equal(t, "prod", snapshot.K8sNamespaceName)
	assert.Equal(t, "app", snapshot.K8sContainerName)
}

func TestPodUIDCandidatesFromCgroupNestedInnermostFirst(t *testing.T) {
	data := []byte("12:cpu,cpuacct:/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-poda7b74024_f357_4d7e_858f_f0da16d1be6f.slice/cri-containerd-b49e204bc952a93405ac65b2ae0092cf0d809b9b900b90a55249ea10af096c7f.scope/kubepods/burstable/pode08bc522-213a-4ed5-85c4-cefbc7623b71/d35e08a556f80149405123b124a22a27339e14c96a8975a8bc49d3594bd12eff\n")
	candidates := podUIDCandidatesFromCgroup(data)
	require.Equal(t, []string{"e08bc522-213a-4ed5-85c4-cefbc7623b71", "a7b74024-f357-4d7e-858f-f0da16d1be6f"}, candidates)
}

func TestContainerCandidatesFromCgroupNestedInnermostFirst(t *testing.T) {
	const inner = "d35e08a556f80149405123b124a22a27339e14c96a8975a8bc49d3594bd12eff"
	const outer = "b49e204bc952a93405ac65b2ae0092cf0d809b9b900b90a55249ea10af096c7f"
	data := []byte("12:cpu,cpuacct:/kubepods.slice/kubepods-burstable.slice/kubepods-burstable-poda7b74024_f357_4d7e_858f_f0da16d1be6f.slice/cri-containerd-" + outer + ".scope/kubepods/burstable/pode08bc522-213a-4ed5-85c4-cefbc7623b71/" + inner + "\n")
	candidates := containerCandidatesFromCgroup(data)
	require.Equal(t, []string{inner, outer}, candidates)

	containerID, podUID := parseCgroupIdentityForPID(data, 123)
	assert.Equal(t, inner, containerID)
	assert.Equal(t, "e08bc522-213a-4ed5-85c4-cefbc7623b71", podUID)
}
