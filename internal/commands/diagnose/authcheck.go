package diagnose

import (
	"embed"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/viper"
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
			Error:        fmt.Sprintf("failed to parse response body: %s", err.Error()),
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

func makeAuthTestRequest(v *viper.Viper) (bool, any, error) {
	collector_url := v.GetString("observe_url")
	authToken := fmt.Sprintf("Bearer %s", v.GetString("token"))
	authTestResponse := makeTestRequest(collector_url, map[string]string{"Authorization": authToken})
	return authTestResponse.Passed, authTestResponse, nil
}

// const networkcheckTemplate = "networkcheck.tmpl"
const authcheckTemplate = "authcheck.tmpl"

var (
	//go:embed authcheck.tmpl
	authcheckTemplateFS embed.FS
)

func authDiagnostic() Diagnostic {
	return Diagnostic{
		check:        makeAuthTestRequest,
		checkName:    "Auth Check",
		templateName: authcheckTemplate,
		templateFS:   authcheckTemplateFS,
	}
}
