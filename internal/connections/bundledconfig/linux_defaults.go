//go:build linux && !docker

package bundledconfig

import (
	"embed"
)

var OverrideTemplates map[string]embed.FS = LinuxTemplateFS
