package mihomo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	baseURL    string
	secret     string
	httpClient *http.Client
}

func NewClient(controllerHostPort, secret string) *Client {
	return &Client{
		baseURL: fmt.Sprintf("http://%s", controllerHostPort),
		secret:  secret,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

func (c *Client) request(method, path string, body interface{}) (*http.Response, error) {
	reqURL := c.baseURL + path
	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, reqURL, bodyReader)
	if err != nil {
		return nil, err
	}
	if c.secret != "" {
		req.Header.Set("Authorization", "Bearer "+c.secret)
	}
	req.Header.Set("Content-Type", "application/json")
	return c.httpClient.Do(req)
}

// GetDelay tests the latency of a named proxy
func (c *Client) GetDelay(proxyName, testURL string, timeoutMs int) (int, error) {
	escapedName := url.PathEscape(proxyName)
	path := fmt.Sprintf("/proxies/%s/delay?url=%s&timeout=%d", escapedName, url.QueryEscape(testURL), timeoutMs)

	resp, err := c.request("GET", path, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("delay test HTTP %d: %s", resp.StatusCode, string(b))
	}

	var res struct {
		Delay   int    `json:"delay"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return 0, err
	}
	if res.Message != "" && res.Delay == 0 {
		return 0, fmt.Errorf("delay error: %s", res.Message)
	}

	return res.Delay, nil
}

// ReloadConfig instructs mihomo to hot reload a new configuration file
func (c *Client) ReloadConfig(configPath string) error {
	payload := map[string]interface{}{
		"path": configPath,
	}
	resp, err := c.request("PUT", "/configs?force=true", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("reload config HTTP %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

// HealthCheck checks if external controller is responding
func (c *Client) HealthCheck() (bool, string) {
	resp, err := c.request("GET", "/version", nil)
	if err != nil {
		return false, err.Error()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Sprintf("status %d", resp.StatusCode)
	}
	var res struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&res)
	return true, res.Version
}
