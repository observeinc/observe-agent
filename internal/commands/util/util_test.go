package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJoinUrl(t *testing.T) {
	testCases := []struct {
		baseURL  string
		path     string
		expected string
	}{
		{"", "", ""},
		{"/", "", ""},
		{"", "/", ""},
		{"", "/path/", "/path"},
		{"localhost:13133", "/status", "localhost:13133/status"},
		{"http://example.com/", "", "http://example.com"},
		{"http://example.com/1", "/2/3", "http://example.com/1/2/3"},
		{"http://example.com/1", "2/3", "http://example.com/1/2/3"},
		{"http://example.com/1/", "2/3", "http://example.com/1/2/3"},
		{"http://example.com/1/", "/2/3", "http://example.com/1/2/3"},
	}
	for _, tc := range testCases {
		result := JoinUrl(tc.baseURL, tc.path)
		assert.Equal(t, tc.expected, result)
	}
}

func TestReplaceEnvString(t *testing.T) {
	t.Setenv("TEST_A", "a")
	t.Setenv("TEST_B", "b")
	t.Setenv("OTHER_VAR", "c")
	testCases := []struct {
		input    string
		expected string
	}{
		{"", ""},
		{"${env:}", ""},
		{"${env:TEST_A}", "a"},
		{"$${env:TEST_A}}", "$a}"},
		{"${env:TEST_A} -- {TEST_B}", "a -- {TEST_B}"},
		{"${env:TEST_A} -- ${env:TEST_B}", "a -- b"},
		{"${env:FAKE} ${env:OTHER_VAR}", " c"},
	}
	for _, tc := range testCases {
		result := ReplaceEnvString(tc.input)
		assert.Equal(t, tc.expected, result)
	}
}
