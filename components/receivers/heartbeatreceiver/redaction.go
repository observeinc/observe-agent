package heartbeatreceiver

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// DefaultPrefixLength is the default number of characters to show before obfuscating
	DefaultPrefixLength = 8
)

// SensitiveFieldPattern defines a pattern for matching and obfuscating sensitive fields in YAML
type SensitiveFieldPattern struct {
	// Path is the YAML path to the field using dot notation
	// Examples: "token", "auth_check.headers.authorization", "database.password"
	// Leave empty if using KeyPattern
	Path string

	// KeyPattern is a regex pattern that matches keys at any depth
	// Example: "^authorization$" will match any field named "authorization" at any level
	// Example: ".*password.*" will match any field containing "password"
	// If both Path and KeyPattern are set, Path takes precedence
	KeyPattern string

	// PrefixLength is the number of characters to show before obfuscating (default: 8)
	PrefixLength int

	// Compiled regex pattern (populated automatically, don't set manually)
	keyRegex *regexp.Regexp
}

// Init compiles the KeyPattern regex if needed. Safe to call multiple times (noop if already initialized).
func (p *SensitiveFieldPattern) Init() error {
	// If already initialized or no pattern to compile, nothing to do
	if p.keyRegex != nil || p.KeyPattern == "" {
		return nil
	}

	// Compile the regex pattern
	re, err := regexp.Compile(p.KeyPattern)
	if err != nil {
		// If the pattern is invalid, treat it as a literal string match
		p.keyRegex = regexp.MustCompile("^" + regexp.QuoteMeta(p.KeyPattern) + "$")
		return fmt.Errorf("invalid regex pattern %q, treating as literal: %w", p.KeyPattern, err)
	}

	p.keyRegex = re
	return nil
}

// Matches checks if the pattern matches the given path and key.
// Returns true if this pattern should be applied to the field.
func (p *SensitiveFieldPattern) Matches(currentPath []string, key string) bool {
	if p.Path != "" {
		// Exact path matching
		patternPath := strings.Split(p.Path, ".")
		return pathsMatch(currentPath, patternPath)
	}

	if p.keyRegex != nil {
		// Regex pattern matching - match if the current key matches the regex
		return p.keyRegex.MatchString(key)
	}

	return false
}

// ApplyObfuscation applies obfuscation to the given value node if it's a scalar.
// Returns true if obfuscation was applied.
func (p *SensitiveFieldPattern) ApplyObfuscation(valueNode *yaml.Node) bool {
	if valueNode.Kind != yaml.ScalarNode {
		return false
	}

	prefixLen := p.PrefixLength
	if prefixLen == 0 {
		prefixLen = DefaultPrefixLength
	}
	valueNode.Value = obfuscateValue(valueNode.Value, prefixLen)
	return true
}

// sensitiveFieldPatterns defines all the sensitive fields that should be obfuscated
var sensitiveFieldPatterns = []SensitiveFieldPattern{
	{
		Path:         "token",
		PrefixLength: 8,
	},
	{
		KeyPattern:   "^[aA]uthorization$",
		PrefixLength: 16,
	},
}

// initSensitiveFieldPatterns compiles all regex patterns
func initSensitiveFieldPatterns() {
	for i := range sensitiveFieldPatterns {
		// Init is a noop if keyRegex is already set or KeyPattern is empty
		_ = sensitiveFieldPatterns[i].Init()
	}
}

// obfuscateValue obfuscates a value by showing a prefix and replacing the rest with asterisks
func obfuscateValue(value string, prefixLength int) string {
	if len(value) > prefixLength {
		return value[:prefixLength] + strings.Repeat("*", len(value)-prefixLength)
	}
	return strings.Repeat("*", len(value))
}

// traverseAndObfuscate recursively traverses a YAML node and obfuscates sensitive fields
func traverseAndObfuscate(node *yaml.Node, currentPath []string, patterns []SensitiveFieldPattern) {
	if node == nil {
		return
	}

	switch node.Kind {
	case yaml.DocumentNode:
		// Document node contains a single content node
		if len(node.Content) > 0 {
			traverseAndObfuscate(node.Content[0], currentPath, patterns)
		}

	case yaml.MappingNode:
		// Mapping nodes have key-value pairs in Content
		// Content is a flat list: [key1, value1, key2, value2, ...]
		for i := 0; i < len(node.Content); i += 2 {
			if i+1 >= len(node.Content) {
				break
			}

			keyNode := node.Content[i]
			valueNode := node.Content[i+1]

			// Build the path for this key
			newPath := append(currentPath, keyNode.Value)

			// Check if this path matches any sensitive field pattern and apply obfuscation
			for _, pattern := range patterns {
				if pattern.Matches(newPath, keyNode.Value) {
					pattern.ApplyObfuscation(valueNode)
					// Don't break - continue checking other patterns in case of multiple matches
				}
			}

			// Recurse into the value node
			traverseAndObfuscate(valueNode, newPath, patterns)
		}

	case yaml.SequenceNode:
		// Sequence nodes contain list items
		for _, item := range node.Content {
			traverseAndObfuscate(item, currentPath, patterns)
		}

	case yaml.ScalarNode:
		// Scalar nodes are leaf values - nothing to traverse
		return
	}
}

// pathsMatch checks if the current path matches the pattern path
func pathsMatch(current []string, pattern []string) bool {
	if len(current) != len(pattern) {
		return false
	}
	for i := range pattern {
		if current[i] != pattern[i] {
			return false
		}
	}
	return true
}

// redactAndEncodeConfig parses YAML config from env var, redacts sensitive fields, and returns base64 encoded string
func redactAndEncodeConfig(yamlContent string) (string, error) {
	if yamlContent == "" {
		return "", fmt.Errorf("empty config content")
	}

	// Parse the YAML content
	var node yaml.Node
	err := yaml.Unmarshal([]byte(yamlContent), &node)
	if err != nil {
		return "", fmt.Errorf("failed to parse YAML: %w", err)
	}

	// Redact sensitive fields
	traverseAndObfuscate(&node, []string{}, sensitiveFieldPatterns)

	// Marshal back to YAML
	redactedYaml, err := yaml.Marshal(&node)
	if err != nil {
		return "", fmt.Errorf("failed to marshal redacted YAML: %w", err)
	}

	// Base64 encode
	encoded := base64.StdEncoding.EncodeToString(redactedYaml)
	return encoded, nil
}
