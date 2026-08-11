package processdiscoveryreceiver

import (
	"debug/dwarf"
	"errors"
	"io"
)

func detectDWARFLanguage(data *dwarf.Data, enabled map[string]struct{}) (string, bool) {
	reader := data.Reader()
	for {
		entry, err := reader.Next()
		if errors.Is(err, io.EOF) || entry == nil {
			return "", false
		}
		if err != nil {
			return "", false
		}
		if entry.Tag != dwarf.TagCompileUnit {
			continue
		}
		language, ok := dwarfLanguageValue(entry.Val(dwarf.AttrLanguage))
		if !ok {
			continue
		}
		runtimeName := dwarfRuntime(language)
		if _, ok := enabled[runtimeName]; ok {
			return runtimeName, true
		}
	}
}

func dwarfLanguageValue(value any) (int64, bool) {
	switch language := value.(type) {
	case int64:
		return language, true
	case uint64:
		return int64(language), true
	case int:
		return int64(language), true
	case uint:
		return int64(language), true
	default:
		return 0, false
	}
}

func dwarfRuntime(language int64) string {
	switch language {
	case 0x0004, 0x0019, 0x001a, 0x0021, 0x002a:
		return "cpp"
	case 0x001c:
		return "rust"
	default:
		return ""
	}
}
