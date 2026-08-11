//go:build linux

package processdiscoveryreceiver

import (
	"debug/buildinfo"
	"debug/elf"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"syscall"
)

func inspectLinuxExecutable(path string, enabled map[string]struct{}) (detectionResult, bool, executableIdentity, error) {
	file, err := os.Open(path)
	if err != nil {
		return detectionResult{}, false, executableIdentity{}, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return detectionResult{}, false, executableIdentity{}, err
	}
	sys, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return detectionResult{}, false, executableIdentity{}, fmt.Errorf("unsupported executable stat type")
	}
	elfFile, err := elf.NewFile(file)
	if err != nil {
		return detectionResult{}, false, executableIdentity{}, err
	}
	defer elfFile.Close()
	identity := executableIdentity{
		Device: uint64(sys.Dev), Inode: sys.Ino, Architecture: elfFile.Machine.String(),
		GNUBuildID: readGNUBuildID(elfFile),
	}

	if _, ok := enabled["go"]; ok {
		if info, err := buildinfo.Read(file); err == nil && info.GoVersion != "" {
			identity.GoBuildID = readGoBuildID(elfFile)
			return detectionResult{
				runtimeFamily: "go", inferredVersion: info.GoVersion, inferredVersionKind: "go_toolchain",
				assertions: []string{"go.buildinfo"},
			}, true, identity, nil
		}
		if elfFile.Section(".gopclntab") != nil {
			return detectionResult{runtimeFamily: "go", assertions: []string{"go.pclntab"}}, true, identity, nil
		}
	}
	if _, ok := enabled["rust"]; ok && hasELFRustSymbols(elfFile) {
		return detectionResult{runtimeFamily: "native", assertions: []string{"rust.symbols"}}, true, identity, nil
	}
	if data, err := elfFile.DWARF(); err == nil {
		if runtimeName, ok := detectDWARFLanguage(data, enabled); ok && runtimeName != "cpp" {
			return detectionResult{runtimeFamily: runtimeName, assertions: []string{runtimeName + ".dwarf_language"}}, true, identity, nil
		}
	}
	return detectionResult{}, false, identity, nil
}

func linuxExecutableIdentity(path string) (executableIdentity, error) {
	file, err := os.Open(path)
	if err != nil {
		return executableIdentity{}, err
	}
	defer file.Close()
	stat, err := file.Stat()
	if err != nil {
		return executableIdentity{}, err
	}
	sys, ok := stat.Sys().(*syscall.Stat_t)
	if !ok {
		return executableIdentity{}, fmt.Errorf("unsupported executable stat type")
	}
	elfFile, err := elf.NewFile(file)
	if err != nil {
		return executableIdentity{}, err
	}
	defer elfFile.Close()
	return executableIdentity{
		Device: uint64(sys.Dev), Inode: sys.Ino, Architecture: elfFile.Machine.String(),
		Size: stat.Size(), ModifiedUnix: stat.ModTime().UnixNano(), GNUBuildID: readGNUBuildID(elfFile),
		GoBuildID: readGoBuildID(elfFile),
	}, nil
}

func readGNUBuildID(file *elf.File) string {
	section := file.Section(".note.gnu.build-id")
	if section == nil {
		return ""
	}
	data, err := section.Data()
	if err != nil || len(data) < 16 {
		return ""
	}
	namesz := file.ByteOrder.Uint32(data[0:4])
	descsz := file.ByteOrder.Uint32(data[4:8])
	descOffset := 12 + (int(namesz)+3)&^3
	if descOffset < 0 || descOffset+int(descsz) > len(data) {
		return ""
	}
	return hex.EncodeToString(data[descOffset : descOffset+int(descsz)])
}

func readGoBuildID(file *elf.File) string {
	section := file.Section(".note.go.buildid")
	if section == nil {
		return ""
	}
	data, err := section.Data()
	if err != nil {
		return ""
	}
	return strings.Trim(string(data), "\x00\x04\xff Go")
}

func hasELFRustSymbols(file *elf.File) bool {
	symbols, _ := file.Symbols()
	dynamicSymbols, _ := file.DynamicSymbols()
	for _, symbol := range append(symbols, dynamicSymbols...) {
		name := strings.ToLower(symbol.Name)
		if strings.Contains(name, "rust_panic") || strings.Contains(name, "rust_eh_personality") {
			return true
		}
	}
	return false
}
