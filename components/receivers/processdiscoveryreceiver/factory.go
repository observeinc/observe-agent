package processdiscoveryreceiver

import (
	"context"
	"sync"

	"github.com/observeinc/observe-agent/components/receivers/processdiscoveryreceiver/internal/metadata"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		metadata.Type,
		createDefaultConfig,
		receiver.WithLogs(createLogsReceiver, metadata.LogsStability),
	)
}

var sharedReceivers = struct {
	sync.Mutex
	items map[component.ID]*processDiscoveryReceiver
}{items: make(map[component.ID]*processDiscoveryReceiver)}

func createLogsReceiver(
	_ context.Context,
	set receiver.Settings,
	cfg component.Config,
	nextConsumer consumer.Logs,
) (receiver.Logs, error) {
	r, err := getSharedReceiver(set, cfg.(*Config))
	if err != nil {
		return nil, err
	}
	r.nextConsumer = nextConsumer
	return r.addReference(), nil
}

func getSharedReceiver(set receiver.Settings, cfg *Config) (*processDiscoveryReceiver, error) {
	sharedReceivers.Lock()
	defer sharedReceivers.Unlock()
	if existing := sharedReceivers.items[set.ID]; existing != nil {
		return existing, nil
	}
	r, err := newReceiver(set, cfg)
	if err != nil {
		return nil, err
	}
	sharedReceivers.items[set.ID] = r
	return r, nil
}
