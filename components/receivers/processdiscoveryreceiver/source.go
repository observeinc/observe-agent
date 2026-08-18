package processdiscoveryreceiver

import (
	"context"
	"time"
)

type ProcessKey struct {
	PID          int32
	StartTime    uint64
	CreationTime time.Time
}

type ProcessIdentity struct {
	Key ProcessKey
}

type ProcessDescription struct {
	ParentPID             int32
	GroupLeaderPID        int32
	SessionLeaderPID      int32
	VirtualPID            int32
	Command               string
	CommandArgs           []string
	ExecutableName        string
	ExecutablePath        string
	WorkingDirectory      string
	Owner                 string
	UserID                int64
	State                 string
	Cgroup                string
	GNUBuildID            string
	GoBuildID             string
	ApplicationEntrypoint string
	ExecutableArch        string
}

type RuntimeEvidence struct {
	RuntimeName         string
	RuntimeFamily       string
	RuntimeVersion      string
	InferredVersion     string
	InferredVersionKind string
	RuntimeAssertions   []string
	RuntimeStatus       string
}

type ObservationProvenance struct {
	Source     string
	ObservedAt time.Time
	CgroupID   uint64
	Complete   bool
}

type WorkloadContext struct {
	ContainerID           string
	ContainerIDCandidates []string
	ContainerName         string
	ContainerRuntime      string
	ContainerImageName    string
	ContainerImageID      string
	K8sNamespaceName      string
	K8sNodeName           string
	K8sPodUID             string
	PodUIDCandidates      []string
	K8sPodName            string
	K8sContainerName      string
	K8sWorkloadKind       string
	K8sWorkloadName       string
	K8sWorkloadUID        string
	K8sCorrelationStatus  string
}

type OTLPConnection struct {
	RemoteHost  string
	RemotePort  int
	Transport   string
	ObservedAt  time.Time
	MatchedRule string
}

type InstrumentationEvidence struct {
	OTLPConnections []OTLPConnection
	Status          string // "detected", "none_detected", "inaccessible", "unavailable"
}

type ProcessSnapshot struct {
	ProcessIdentity
	ProcessDescription
	RuntimeEvidence
	ObservationProvenance
	WorkloadContext
	InstrumentationEvidence
	ArgsCount int
}

type ProcessRelationship struct {
	Type       string
	Attributes map[string]string
}

type ProcessObservation struct {
	Process       ProcessSnapshot
	Relationships []ProcessRelationship
	ObservedAt    time.Time
}

type ScanResult struct {
	Processes       []ProcessSnapshot
	UnavailablePIDs map[int32]struct{}
	Complete        bool
}

type ProcessSource interface {
	Scan(context.Context) (ScanResult, error)
	Inspect(context.Context, int32) (ProcessSnapshot, error)
}

type LifecycleEventType string

const (
	lifecycleExec LifecycleEventType = "exec"
	lifecycleExit LifecycleEventType = "exit"
	lifecycleLoss LifecycleEventType = "loss"
)

type LifecycleEvent struct {
	Type       LifecycleEventType
	PID        int32
	ParentPID  int32
	ObservedAt time.Time
	CgroupID   uint64
	Lost       uint64
	FD         int32
	ThreadID   int32
}

type LifecycleSource interface {
	Start(context.Context) (<-chan LifecycleEvent, error)
	Close() error
}

type processEmitter interface {
	Emit(context.Context, []processEvent) error
}

type disabledLifecycleSource struct{}

func (disabledLifecycleSource) Start(context.Context) (<-chan LifecycleEvent, error) { return nil, nil }
func (disabledLifecycleSource) Close() error                                         { return nil }

func mergeDetectionResult(result *detectionResult, found *bool, candidate detectionResult) {
	if !*found {
		*result = candidate
		*found = true
		return
	}
	if result.runtimeFamily == "native" && candidate.runtimeFamily != "" && candidate.runtimeFamily != "native" {
		candidate.assertions = appendUnique(candidate.assertions, result.assertions...)
		*result = candidate
		return
	}
	if candidate.runtimeFamily != "" && candidate.runtimeFamily != "native" &&
		result.runtimeFamily != "" && candidate.runtimeFamily != result.runtimeFamily {
		return
	}
	result.assertions = appendUnique(result.assertions, candidate.assertions...)
	if result.inferredVersion == "" && candidate.inferredVersion != "" &&
		(candidate.runtimeFamily == "" || candidate.runtimeFamily == result.runtimeFamily) {
		result.inferredVersion = candidate.inferredVersion
		result.inferredVersionKind = candidate.inferredVersionKind
	}
}

func appendUnique(values []string, additions ...string) []string {
	for _, addition := range additions {
		found := false
		for _, value := range values {
			if value == addition {
				found = true
				break
			}
		}
		if !found {
			values = append(values, addition)
		}
	}
	return values
}
