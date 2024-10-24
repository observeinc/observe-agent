package sendtestdata

import (
	"testing"

	"github.com/jarcoal/httpmock"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

func TestPostTestData(t *testing.T) {
	httpmock.Activate()
	t.Cleanup(httpmock.DeactivateAndReset)

	expectedResponse := `{"ok":true}`
	// Verify that the data is sent to the expected endpoint along with the bearer and json headers.
	httpmock.RegisterMatcherResponder("POST", "https://123456.collect.observe-eng.com/v1/http/test",
		httpmock.BodyContainsString(`"hello":"world"`).And(
			httpmock.HeaderIs("Content-Type", "application/json"),
			httpmock.HeaderIs("Authorization", "Bearer test-token"),
		),
		httpmock.NewStringResponder(200, expectedResponse),
	)

	viper.Set("observe_url", "https://123456.collect.observe-eng.com/")
	viper.Set("token", "test-token")
	testData := map[string]string{"hello": "world"}
	resp, err := PostTestDataToObserve(testData, "/test")
	assert.NoError(t, err)
	assert.Equal(t, expectedResponse, resp)
}
