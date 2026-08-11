package processdiscoveryreceiver

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArgumentScrubberRedactsSensitiveValues(t *testing.T) {
	scrubber := newArgumentScrubber(CommandLineConfig{MaxArguments: 10})
	assert.Equal(t, []string{
		"app", "--password", redactedArgument, "--api_key=" + redactedArgument, "safe",
	}, scrubber.scrub([]string{"app", "--password", "hunter2", "--api_key=abc", "safe"}))
}

func TestArgumentScrubberSupportsCustomWildcards(t *testing.T) {
	scrubber := newArgumentScrubber(CommandLineConfig{MaxArguments: 10, CustomSensitiveWords: []string{"*token"}})
	assert.Equal(t, []string{"app", "--session_token=" + redactedArgument}, scrubber.scrub([]string{"app", "--session_token=abc"}))
}

func TestArgumentScrubberStripsAndBoundsArguments(t *testing.T) {
	scrubber := newArgumentScrubber(CommandLineConfig{MaxArguments: 2, StripAllArguments: true})
	assert.Equal(t, []string{"app", redactedArgument}, scrubber.scrub([]string{"app", "one", "two"}))
}
