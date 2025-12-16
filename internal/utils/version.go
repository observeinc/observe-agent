package utils

import "github.com/observeinc/observe-agent/build"

// GetAgentVersion returns the current agent version.
// Returns "dev" if the version is not set (development builds).
func GetAgentVersion() string {
	if build.Version == "" {
		return "dev"
	}
	return build.Version
}

