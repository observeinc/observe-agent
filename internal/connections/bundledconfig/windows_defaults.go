//go:build windows

package bundledconfig

import (
	"embed"
)

var OverrideTemplates map[string]embed.FS = WindowsTemplateFS
