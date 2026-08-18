//go:build linux

package processdiscoveryreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNextPowerOfTwo(t *testing.T) {
	assert.Equal(t, 4096, nextPowerOfTwo(3000))
	assert.Equal(t, 1, nextPowerOfTwo(1))
	assert.Equal(t, 16, nextPowerOfTwo(9))
}

func TestLifecycleRecordDoesNotClaimParentPID(t *testing.T) {
	record := bpfLifecycleRecord{PID: 10}
	assert.Zero(t, record.ParentPID)
}
