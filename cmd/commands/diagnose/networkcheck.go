package diagnose

import (
	"net/http"
)

const (
	ChallengeURL = "https://175914298205.collect.observeinc.com/.well-known/fastly/logging/challenge"
)

type NetworkTestResult struct {
	URL          string
	ResponseCode int
	Passed       bool
	Error        error
}

func MakeTestRequest(URL string) NetworkTestResult {
	resp, err := http.Get(URL)
	if err != nil {
		return NetworkTestResult{
			Passed:       false,
			Error:        err,
			ResponseCode: resp.StatusCode,
			URL:          URL,
		}
	}
	return NetworkTestResult{
		Passed:       resp.StatusCode == 200,
		Error:        nil,
		ResponseCode: resp.StatusCode,
		URL:          URL,
	}
}
