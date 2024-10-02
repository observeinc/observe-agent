package diagnose

import (
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/viper"
)

const (
	ChallengeURL = "https://175914298205.collect.observeinc.com/.well-known/fastly/logging/challenge"
	AuthCheckURL = "https://175914298205.collect.observeinc.com/status"
)

type NetworkTestResult struct {
	URL          string
	ResponseCode int
	Passed       bool
	Error        string
}

func makeTestRequest(URL string, headers map[string]string) NetworkTestResult {
	client := &http.Client{}
	req, err := http.NewRequest("GET", URL, nil)
	if err != nil {
		return NetworkTestResult{
			Passed:       false,
			Error:        err.Error(),
			ResponseCode: 0,
			URL:          URL,
		}
	}
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return NetworkTestResult{
			Passed:       false,
			Error:        err.Error(),
			ResponseCode: 0,
			URL:          URL,
		}
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return NetworkTestResult{
			Passed:       false,
			Error:        "failed to parse response body",
			ResponseCode: resp.StatusCode,
			URL:          URL,
		}
	}
	bodyString := string(bodyBytes)
	if resp.StatusCode != 200 {
		return NetworkTestResult{
			Passed:       false,
			Error:        bodyString,
			ResponseCode: resp.StatusCode,
			URL:          URL,
		}
	}
	return NetworkTestResult{
		Passed:       true,
		ResponseCode: resp.StatusCode,
		URL:          URL,
	}
}

func makeNetworkingTestRequest(url string) NetworkTestResult {
	return makeTestRequest(url, make(map[string]string))
}

func makeAuthTestRequest(url string) NetworkTestResult {
	authToken := fmt.Sprintf("Bearer %s", viper.GetString("token"))
	return makeTestRequest(url, map[string]string{"Authorization": authToken})
}
