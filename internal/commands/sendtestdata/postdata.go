package sendtestdata

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/viper"
)

func PostTestData(data any, URL string, headers map[string]string) (string, error) {
	postBody, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	client := &http.Client{}
	req, err := http.NewRequest("POST", URL, bytes.NewBuffer(postBody))
	if err != nil {
		return "", err
	}
	headers["Content-Type"] = "application/json"
	for key, value := range headers {
		req.Header.Add(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	bodyString := string(bodyBytes)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("sending test data to %s failed with response: %s", URL, bodyString)
	}
	return bodyString, nil
}

func PostDataToObserve(data any, extraPath string, v *viper.Viper) (string, error) {
	collector_url := v.GetString("observe_url")
	endpoint := fmt.Sprintf("%s/v1/http%s", strings.TrimRight(collector_url, "/"), extraPath)
	authToken := fmt.Sprintf("Bearer %s", v.GetString("token"))
	return PostTestData(data, endpoint, map[string]string{"Authorization": authToken})
}
