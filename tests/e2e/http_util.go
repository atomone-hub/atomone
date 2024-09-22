package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

func httpGet(endpoint string) ([]byte, error) {
	return httpGet_(endpoint, 0)
}

const maxAttempt = 5

func isErrNotFound(err error) bool {
	return err != nil && strings.HasSuffix(err.Error(), http.StatusText(http.StatusNotFound))
}

func httpGet_(endpoint string, attempt int) ([]byte, error) {
	resp, err := http.Get(endpoint) //nolint:gosec // this is only used during tests
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusServiceUnavailable && attempt < maxAttempt {
		// node not avail, wait and retry
		time.Sleep(time.Second)
		return httpGet_(endpoint, attempt+1)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server status %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func readJSON(resp *http.Response) (map[string]interface{}, error) {
	defer resp.Body.Close()

	body, readErr := io.ReadAll(resp.Body)
	if readErr != nil {
		return nil, fmt.Errorf("failed to read Body")
	}

	var data map[string]interface{}
	err := json.Unmarshal(body, &data)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal response body")
	}

	return data, nil
}
