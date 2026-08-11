//go:build linux

package processdiscoveryreceiver

import (
	"testing"

	"github.com/cilium/ebpf"
	"github.com/stretchr/testify/assert"
)

func TestLifecycleProgramSpecs(t *testing.T) {
	placeholder := &ebpf.Map{}
	exec := lifecycleProgramSpec("exec", placeholder, 1)
	exit := lifecycleProgramSpec("exit", placeholder, 2)
	assert.Equal(t, ebpf.TracePoint, exec.Type)
	assert.Equal(t, ebpf.TracePoint, exit.Type)
	assert.NotEmpty(t, exec.Instructions)
	assert.NotEmpty(t, exit.Instructions)
}

func TestAcceptProgramSpec(t *testing.T) {
	program := acceptProgramSpec(&ebpf.Map{}, 3, false)
	assert.Equal(t, ebpf.TracePoint, program.Type)
	assert.NotEmpty(t, program.Instructions)
}

func TestNextPowerOfTwo(t *testing.T) {
	assert.Equal(t, 4096, nextPowerOfTwo(3000))
}

func TestLifecycleRecordDoesNotClaimParentPID(t *testing.T) {
	record := bpfLifecycleRecord{PID: 10}
	assert.Zero(t, record.ParentPID)
}
