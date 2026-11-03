package configconverter

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
)

// fileStorageKinds match the names filestorage.kindString emits.
var fileStorageKinds = []string{"receiver", "processor", "exporter", "connector", "extension"}

// fileStorageOwnerSuffixes are the extra GetClient name arguments used by
// persistent exporter queues (the signal). An empty suffix covers receivers
// such as filelog, which call GetClient with an empty name.
var fileStorageOwnerSuffixes = []string{"", "logs", "metrics", "traces"}

// MigrateFileStorage renames on-disk file-storage databases whose owner IDs
// were remapped by LegacyComponentIDs. Without this, a rename of e.g.
// otlphttp/observe to otlp_http/observe leaves exporter_otlphttp_observe_logs
// undrained while the new exporter opens an empty exporter_otlp_http_observe_logs.
//
// Existing destination files are left alone so a queue that already started
// under the new ID is not overwritten. Missing sources are ignored.
func MigrateFileStorage(dir string, log *zap.Logger) error {
	if dir == "" {
		return nil
	}
	if log == nil {
		log = zap.NewNop()
	}
	info, err := os.Stat(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("file storage path %q is not a directory", dir)
	}

	for legacy, canonical := range LegacyComponentIDs {
		legacyType, legacyName := splitComponentID(legacy)
		canonType, canonName := splitComponentID(canonical)
		for _, kind := range fileStorageKinds {
			for _, suffix := range fileStorageOwnerSuffixes {
				src := filepath.Join(dir, fileStorageFilename(kind, legacyType, legacyName, suffix))
				dst := filepath.Join(dir, fileStorageFilename(kind, canonType, canonName, suffix))
				if err := renameFileStorageIfNeeded(src, dst, log); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func renameFileStorageIfNeeded(src, dst string, log *zap.Logger) error {
	if _, err := os.Stat(src); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if _, err := os.Stat(dst); err == nil {
		log.Warn("file storage already has a database for the new component id; leaving the legacy file in place",
			zap.String("legacy_file", src),
			zap.String("canonical_file", dst),
		)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(src, dst); err != nil {
		return err
	}
	log.Info("renamed file storage database to match a remapped component id",
		zap.String("from", src),
		zap.String("to", dst),
	)
	return nil
}

func splitComponentID(id string) (typ, name string) {
	typ, name, _ = strings.Cut(id, "/")
	return typ, name
}

func fileStorageFilename(kind, typ, name, suffix string) string {
	var raw string
	if suffix == "" {
		raw = fmt.Sprintf("%s_%s_%s", kind, typ, name)
	} else {
		raw = fmt.Sprintf("%s_%s_%s_%s", kind, typ, name, suffix)
	}
	return sanitizeFileStorageName(raw)
}

// sanitizeFileStorageName mirrors the contrib filestorage sanitizer so generated
// names match files already on disk. Safe characters are left unchanged;
// everything else (including '~') becomes ~XXXX.
func sanitizeFileStorageName(name string) string {
	var b strings.Builder
	for _, c := range name {
		if isFileStorageSafe(c) {
			b.WriteRune(c)
			continue
		}
		fmt.Fprintf(&b, "~%04X", c)
	}
	return b.String()
}

func isFileStorageSafe(c rune) bool {
	switch {
	case c >= 'a' && c <= 'z',
		c >= 'A' && c <= 'Z',
		c >= '0' && c <= '9',
		c == '.',
		c == '-',
		c == '_':
		return true
	}
	return false
}
