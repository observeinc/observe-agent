package utils

import (
	"os"
	"runtime"
)

// GetDefaultAgentPath returns the default path for agent configuration and data files
// based on the operating system.
func GetDefaultAgentPath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return "/usr/local/observe-agent"
	case "windows":
		return os.ExpandEnv("$ProgramFiles\\Observe\\observe-agent")
	case "linux":
		return "/etc/observe-agent"
	default:
		return "/etc/observe-agent"
	}
}