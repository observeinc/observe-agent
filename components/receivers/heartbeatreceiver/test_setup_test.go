package heartbeatreceiver

import (
	"os"
)

// init sets up default environment variable for tests if not already set
func init() {
	if os.Getenv("OBSERVE_AGENT_INSTANCE_ID") == "" {
		os.Setenv("OBSERVE_AGENT_INSTANCE_ID", "test-agent-default-id")
	}
}