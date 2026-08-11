package processdiscoveryreceiver

import (
	"path/filepath"
	"strings"
)

const redactedArgument = "<redacted>"

type argumentScrubber struct {
	patterns []string
	stripAll bool
	maxArgs  int
}

func newArgumentScrubber(cfg CommandLineConfig) argumentScrubber {
	patterns := append([]string{}, defaultSensitiveWords...)
	patterns = append(patterns, cfg.CustomSensitiveWords...)
	for index := range patterns {
		patterns[index] = strings.ToLower(patterns[index])
	}
	return argumentScrubber{patterns: patterns, stripAll: cfg.StripAllArguments, maxArgs: cfg.MaxArguments}
}

func (s argumentScrubber) scrub(args []string) []string {
	if len(args) > s.maxArgs {
		args = args[:s.maxArgs]
	}
	if len(args) == 0 {
		return nil
	}
	out := append([]string(nil), args...)
	if s.stripAll {
		for index := 1; index < len(out); index++ {
			out[index] = redactedArgument
		}
		return out
	}
	redactNext := false
	for index, argument := range out {
		if redactNext {
			out[index] = redactedArgument
			redactNext = false
			continue
		}
		key, _, hasValue := strings.Cut(argument, "=")
		key = strings.TrimLeft(strings.ToLower(key), "-")
		if !s.sensitive(key) {
			continue
		}
		if hasValue {
			out[index] = argument[:strings.Index(argument, "=")+1] + redactedArgument
		} else {
			redactNext = true
		}
	}
	return out
}

func (s argumentScrubber) sensitive(key string) bool {
	for _, pattern := range s.patterns {
		if matched, _ := filepath.Match(pattern, key); matched || key == pattern {
			return true
		}
	}
	return false
}
