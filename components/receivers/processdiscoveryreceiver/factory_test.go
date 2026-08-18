package processdiscoveryreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
)

func TestFactory(t *testing.T) {
	factory := NewFactory()
	assert.Equal(t, component.MustNewType("processdiscovery"), factory.Type())
	require.NoError(t, componenttest.CheckConfigStruct(factory.CreateDefaultConfig()))
	assert.Equal(t, component.StabilityLevelDevelopment, factory.LogsStability())
}
