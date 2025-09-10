package macos

import "embed"

var (
	//go:embed fleet/heartbeat_receiver.yaml.tmpl
	HeartbeatTemplateFS embed.FS
)

// Add Mac specific templates here if we ever need them
