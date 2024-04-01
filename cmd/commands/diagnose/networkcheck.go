package diagnose

import (
	"fmt"
	"io"
	"net/http"
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

func makeTestRequest(URL string) NetworkTestResult {
	resp, err := http.Get(URL)
	if err != nil {
		return NetworkTestResult{
			Passed:       false,
			Error:        err.Error(),
			ResponseCode: resp.StatusCode,
			URL:          URL,
		}
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body: ", err)
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

func makeNetworkingTestRequest() NetworkTestResult {
	return makeTestRequest(ChallengeURL)
}

func makeAuthTestRequest() NetworkTestResult {
	return makeTestRequest(AuthCheckURL)
}
