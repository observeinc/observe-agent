package heartbeatreceiver

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeAuthRequest(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse int
		serverBody     string
		authHeader     string
		expectedPassed bool
		expectedError  string
	}{
		{
			name:           "successful auth request",
			serverResponse: 200,
			serverBody:     "OK",
			authHeader:     "Bearer test-token",
			expectedPassed: true,
			expectedError:  "",
		},
		{
			name:           "unauthorized request",
			serverResponse: 401,
			serverBody:     "Unauthorized",
			authHeader:     "Bearer invalid-token",
			expectedPassed: false,
			expectedError:  "Unauthorized",
		},
		{
			name:           "server error",
			serverResponse: 500,
			serverBody:     "Internal Server Error",
			authHeader:     "Bearer test-token",
			expectedPassed: false,
			expectedError:  "Internal Server Error",
		},
		{
			name:           "no auth header",
			serverResponse: 200,
			serverBody:     "OK",
			authHeader:     "",
			expectedPassed: true,
			expectedError:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Check if auth header is present when expected
				if tt.authHeader != "" {
					authHeader := r.Header.Get("Authorization")
					assert.Equal(t, tt.authHeader, authHeader)
				}

				w.WriteHeader(tt.serverResponse)
				w.Write([]byte(tt.serverBody))
			}))
			defer server.Close()

			// Make the request
			result := makeAuthRequest(server.URL, tt.authHeader)

			// Verify results
			assert.Equal(t, tt.expectedPassed, result.Passed)
			assert.Equal(t, tt.serverResponse, result.ResponseCode)
			assert.Equal(t, server.URL, result.URL)

			if tt.expectedError != "" {
				assert.Equal(t, tt.expectedError, result.Error)
			}
		})
	}
}

func TestPerformAuthCheck(t *testing.T) {
	tests := []struct {
		name                string
		collectorURL        string
		authHeader          string
		expectedPassed      bool
		expectedErrorSubstr string
	}{
		{
			name:                "missing collector URL",
			collectorURL:        "",
			authHeader:          "Bearer test-token",
			expectedPassed:      false,
			expectedErrorSubstr: "OBSERVE_COLLECTOR_URL environment variable is not set",
		},
		{
			name:                "missing auth header",
			collectorURL:        "https://example.com",
			authHeader:          "",
			expectedPassed:      false,
			expectedErrorSubstr: "OBSERVE_AUTHORIZATION_HEADER environment variable is not set",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment variables
			if tt.collectorURL != "" {
				os.Setenv("OBSERVE_COLLECTOR_URL", tt.collectorURL)
			} else {
				os.Unsetenv("OBSERVE_COLLECTOR_URL")
			}

			if tt.authHeader != "" {
				os.Setenv("OBSERVE_AUTHORIZATION_HEADER", tt.authHeader)
			} else {
				os.Unsetenv("OBSERVE_AUTHORIZATION_HEADER")
			}

			// Perform the check
			result := PerformAuthCheck()

			// Verify results
			assert.Equal(t, tt.expectedPassed, result.Passed)
			if tt.expectedErrorSubstr != "" {
				assert.Contains(t, result.Error, tt.expectedErrorSubstr)
			}

			// Clean up environment variables
			os.Unsetenv("OBSERVE_COLLECTOR_URL")
			os.Unsetenv("OBSERVE_AUTHORIZATION_HEADER")
		})
	}
}

func TestPerformAuthCheckWithValidServer(t *testing.T) {
	// Create test server that returns 200
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		assert.Equal(t, "Bearer test-token", authHeader)
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Set environment variables
	os.Setenv("OBSERVE_COLLECTOR_URL", server.URL)
	os.Setenv("OBSERVE_AUTHORIZATION_HEADER", "Bearer test-token")
	defer func() {
		os.Unsetenv("OBSERVE_COLLECTOR_URL")
		os.Unsetenv("OBSERVE_AUTHORIZATION_HEADER")
	}()

	// Perform the check
	result := PerformAuthCheck()

	// Verify results
	assert.True(t, result.Passed)
	assert.Equal(t, 200, result.ResponseCode)
	assert.Equal(t, server.URL, result.URL)
	assert.Empty(t, result.Error)
}
