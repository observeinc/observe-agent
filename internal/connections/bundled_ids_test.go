package connections

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/observeinc/observe-agent/internal/configconverter"
	"github.com/observeinc/observe-agent/internal/connections/bundledconfig"
	"github.com/stretchr/testify/require"
)

func TestBundledTemplatesUseCanonicalComponentIDs(t *testing.T) {
	sets := []bundledconfig.ConfigTemplates{
		bundledconfig.SharedTemplateFS,
		bundledconfig.LinuxTemplateFS,
		bundledconfig.DockerTemplateFS,
		bundledconfig.WindowsTemplateFS,
		bundledconfig.MacOSTemplateFS,
	}
	for _, templates := range sets {
		for name, efs := range templates {
			body, err := fs.ReadFile(efs, fsRootFile(efs))
			require.NoError(t, err, name)
			text := string(body)
			for legacy := range configconverter.LegacyComponentIDs {
				if strings.Contains(text, legacy) {
					t.Errorf("%s still uses legacy component id %q; bundled templates must use canonical ids so later overrides keep precedence", name, legacy)
				}
			}
		}
	}
}

func fsRootFile(efs fs.FS) string {
	var found string
	_ = fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		found = path
		return fs.SkipAll
	})
	return found
}
