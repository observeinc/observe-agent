//go:build darwin

package bundledconfig

import (
	"embed"
)

var OverrideTemplates map[string]embed.FS = MacOSTemplateFS
var ConfigEnvironment = "macos"
