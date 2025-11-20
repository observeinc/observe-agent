package util

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

// JoinUrl joins a base URL and a path. The resulting URL will never have a trailing slash.
func JoinUrl(baseURL string, path string) string {
	var result string
	if len(baseURL) == 0 {
		result = path
	} else {
		result = fmt.Sprintf("%s/%s", strings.TrimRight(baseURL, "/"), strings.TrimLeft(path, "/"))
	}
	return strings.TrimRight(result, "/")
}

var envRe *regexp.Regexp = regexp.MustCompile(`\${env:\w*?}`)

// ReplaceEnvString replaces all ${env:ENV_VAR} strings in the input string with the value of the environment variable.
func ReplaceEnvString(input string) string {
	return envRe.ReplaceAllStringFunc(input, func(s string) string {
		// The index transforms `${env:ENV_VAR}` to `ENV_VAR`.
		return os.Getenv(s[6 : len(s)-1])
	})
}
