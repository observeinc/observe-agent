package processdiscoveryreceiver

import (
	"path/filepath"
	"regexp"
	"strings"
)

var (
	pythonExecutable = regexp.MustCompile(`^(python(?:\d+(?:\.\d+)*)?|pypy(?:\d+)?|py)$`)
	rubyExecutable   = regexp.MustCompile(`^(ruby(?:\d+(?:\.\d+)*)?|rubyw)$`)
	phpExecutable    = regexp.MustCompile(`^(php(?:-cgi|-fpm)?(?:\d+(?:\.\d+)*)?)$`)
	perlExecutable   = regexp.MustCompile(`^(perl(?:\d+(?:\.\d+)*)?)$`)
)

type detectionResult struct {
	runtimeName         string
	runtimeFamily       string
	runtimeVersion      string
	inferredVersion     string
	inferredVersionKind string
	assertions          []string
}

type runtimeDetector struct {
	enabled map[string]struct{}
}

func newRuntimeDetector(runtimes []string) runtimeDetector {
	enabled := make(map[string]struct{}, len(runtimes))
	for _, runtimeName := range runtimes {
		enabled[runtimeName] = struct{}{}
	}
	return runtimeDetector{enabled: enabled}
}

func (d runtimeDetector) detectCommand(command string, args []string) (detectionResult, bool) {
	command = normalizeCommand(command)
	if d.isEnabled("ruby") && javaMainTarget(args) == "org.jruby.Main" {
		return detectionResult{runtimeName: "jruby", runtimeFamily: "ruby", assertions: []string{"ruby.jruby_main"}}, true
	}
	result := detectionResult{}
	switch {
	case d.isEnabled("java") && command == "java":
		result.runtimeFamily, result.assertions = "java", []string{"java.executable"}
	case d.isEnabled("python") && pythonExecutable.MatchString(command):
		result.runtimeFamily, result.assertions = "python", []string{"python.executable"}
		result.inferredVersion, result.inferredVersionKind = versionFromCommand(command, "python", "pypy"), "executable_suffix"
	case d.isEnabled("nodejs") && (command == "node" || command == "nodejs"):
		result.runtimeFamily, result.assertions = "nodejs", []string{"node.executable"}
	case d.isEnabled("dotnet") && command == "dotnet":
		result.runtimeFamily, result.assertions = "dotnet", []string{"dotnet.executable"}
	case d.isEnabled("ruby") && rubyExecutable.MatchString(command):
		result.runtimeFamily, result.assertions = "ruby", []string{"ruby.executable"}
		result.inferredVersion, result.inferredVersionKind = versionFromCommand(command, "ruby"), "executable_suffix"
	case d.isEnabled("elixir") && (command == "elixir" || command == "elixir_iex" || command == "iex"):
		result.runtimeFamily, result.assertions = "elixir", []string{"elixir.executable"}
	case d.isEnabled("elixir") && (command == "beam.smp" || command == "beam") && containsArgument(args, "-elixir_root"):
		result.runtimeFamily, result.assertions = "elixir", []string{"elixir.beam_arguments"}
	case d.isEnabled("erlang") && (command == "beam.smp" || command == "beam" || command == "erl" || command == "erlexec"):
		result.runtimeFamily, result.assertions = "erlang", []string{"erlang.executable"}
	case d.isEnabled("php") && phpExecutable.MatchString(command):
		result.runtimeFamily, result.assertions = "php", []string{"php.executable"}
	case d.isEnabled("perl") && perlExecutable.MatchString(command):
		result.runtimeFamily, result.assertions = "perl", []string{"perl.executable"}
	default:
		return detectionResult{}, false
	}
	return result, true
}

func (d runtimeDetector) isEnabled(runtimeName string) bool {
	_, ok := d.enabled[runtimeName]
	return ok
}

func normalizeCommand(command string) string {
	command = strings.Trim(strings.TrimSpace(command), `"'`)
	return strings.ToLower(strings.TrimSuffix(filepath.Base(command), ".exe"))
}

func versionFromCommand(command string, prefixes ...string) string {
	for _, prefix := range prefixes {
		if strings.HasPrefix(command, prefix) {
			return strings.TrimLeft(strings.TrimPrefix(command, prefix), "-_")
		}
	}
	return ""
}

func containsArgument(args []string, wanted string) bool {
	for _, arg := range args {
		if arg == wanted {
			return true
		}
	}
	return false
}

func javaMainTarget(args []string) string {
	if len(args) < 2 {
		return ""
	}
	for _, arg := range args[1:] {
		if !strings.HasPrefix(arg, "-") {
			return arg
		}
	}
	return ""
}
