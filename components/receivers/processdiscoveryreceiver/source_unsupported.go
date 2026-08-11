//go:build !linux

package processdiscoveryreceiver

import "context"

type unsupportedProcessSource struct{}

func (unsupportedProcessSource) Scan(context.Context) (ScanResult, error) {
	return ScanResult{}, nil
}

func (unsupportedProcessSource) Inspect(context.Context, int32) (ProcessSnapshot, error) {
	return ProcessSnapshot{}, nil
}

func newProcessSource(*Config) (ProcessSource, error) {
	return unsupportedProcessSource{}, nil
}

func platformSupported() bool { return false }
