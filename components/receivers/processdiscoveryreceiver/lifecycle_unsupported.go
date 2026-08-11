//go:build !linux

package processdiscoveryreceiver

import "go.uber.org/zap"

func newLifecycleSource(*Config, *zap.Logger) (LifecycleSource, error) {
	return disabledLifecycleSource{}, nil
}
