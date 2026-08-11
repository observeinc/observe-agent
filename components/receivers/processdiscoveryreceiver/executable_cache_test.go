package processdiscoveryreceiver

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutableCacheIdentityAndExpiry(t *testing.T) {
	cache := newExecutableCache(time.Minute, 2)
	now := time.Now()
	key := executableIdentity{Device: 1, Inode: 2, Architecture: "x86_64"}
	cache.put(key, detectionResult{runtimeFamily: "go"}, true, now)
	result, found, ok := cache.get(key, now.Add(time.Second))
	require.True(t, ok)
	assert.True(t, found)
	assert.Equal(t, "go", result.runtimeFamily)
	_, _, ok = cache.get(key, now.Add(2*time.Minute))
	assert.False(t, ok)
}
