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

func GetDefaultAgentDataPath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return "/var/lib/observe-agent/data"
	case "windows":
		return os.ExpandEnv("$ProgramData\\Observe\\observe-agent\\data")
	case "linux":
		return "/var/lib/observe-agent/data"
	default:
		return "/var/lib/observe-agent/data"
	}
}

// GetDefaultFilestoragePath returns the default path for file storage
// based on the operating system.
func GetDefaultFilestoragePath() string {
	switch currOS := runtime.GOOS; currOS {
	case "darwin":
		return "/var/lib/observe-agent/filestorage"
	case "windows":
		return os.ExpandEnv("$ProgramData\\Observe\\observe-agent\\filestorage")
	case "linux":
		return "/var/lib/observe-agent/filestorage"
	default:
		return "/var/lib/observe-agent/filestorage"
	}
}
