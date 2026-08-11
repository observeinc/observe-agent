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

func TestKubernetesEnricherDoesNotGuessMismatchedPod(t *testing.T) {
	enricher := &kubernetesEnricher{byContainer: map[string]podAssociation{
		testContainerID: {containerID: testContainerID, podUID: "expected"},
	}}
	snapshot := ProcessSnapshot{WorkloadContext: WorkloadContext{ContainerID: testContainerID, K8sPodUID: "different"}}
	enricher.Enrich(&snapshot)
	assert.Equal(t, "ambiguous", snapshot.K8sCorrelationStatus)
	assert.Empty(t, snapshot.K8sPodName)
}
