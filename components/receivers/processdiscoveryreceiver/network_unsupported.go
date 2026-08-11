//go:build !linux

package processdiscoveryreceiver

func discoverNetworkEndpoints(string, int32, int) ([]NetworkEndpoint, string) {
	return nil, "unavailable"
}
