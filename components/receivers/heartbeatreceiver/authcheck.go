package heartbeatreceiver

import (
	"fmt"
	"io"
	"net/http"
	"os"
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

// PerformAuthCheck performs an authentication check using environment variables
// OBSERVE_COLLECTOR_URL and OBSERVE_AUTHORIZATION_HEADER
func PerformAuthCheck() AuthCheckResult {
	collectorURL := os.Getenv("OBSERVE_COLLECTOR_URL")
	authHeader := os.Getenv("OBSERVE_AUTHORIZATION_HEADER")

	if collectorURL == "" {
		return AuthCheckResult{
			Passed: false,
			Error:  "Observe url environment variable is not set",
			URL:    "",
		}
	}

	if authHeader == "" {
		return AuthCheckResult{
			Passed: false,
			Error:  "Observe token environment variable is not set",
			URL:    collectorURL,
		}
	}

	return makeAuthRequest(collectorURL, authHeader)
}
