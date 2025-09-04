package heartbeatreceiver

import (
	"fmt"
	"io"
	"net/http"
)

// AuthCheckResult represents the result of an authentication check
type AuthCheckResult struct {
	URL          string
	ResponseCode int
	Passed       bool
	Error        string
}

// makeAuthRequest performs an HTTP GET request to test authentication
func makeAuthRequest(url string, authHeader string) AuthCheckResult {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return AuthCheckResult{
			Passed:       false,
			Error:        err.Error(),
			ResponseCode: 0,
			URL:          url,
		}
	}

	// Add authorization header if provided
	if authHeader != "" {
		req.Header.Add("Authorization", authHeader)
	}

	resp, err := client.Do(req)
	if err != nil {
		return AuthCheckResult{
			Passed:       false,
			Error:        err.Error(),
			ResponseCode: 0,
			URL:          url,
		}
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return AuthCheckResult{
			Passed:       false,
			Error:        fmt.Sprintf("failed to parse response body: %s", err.Error()),
			ResponseCode: resp.StatusCode,
			URL:          url,
		}
	}

	bodyString := string(bodyBytes)
	if resp.StatusCode != 200 {
		return AuthCheckResult{
			Passed:       false,
			Error:        bodyString,
			ResponseCode: resp.StatusCode,
			URL:          url,
		}
	}

	return AuthCheckResult{
		Passed:       true,
		ResponseCode: resp.StatusCode,
		URL:          url,
	}
}

// PerformAuthCheck performs an authentication check using the provided URL and auth header
func PerformAuthCheck(URL, authHeader string) AuthCheckResult {
	if URL == "" {
		return AuthCheckResult{
			Passed: false,
			Error:  "OBSERVE_COLLECTOR_URL environment variable is not set",
			URL:    "",
		}
	}

	if authHeader == "" {
		return AuthCheckResult{
			Passed: false,
			Error:  "OBSERVE_AUTHORIZATION_HEADER environment variable is not set",
			URL:    URL,
		}
	}

	return makeAuthRequest(URL, authHeader)
}
