package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	DefaultBaseURL = "https://api.sockt.dev"
	DefaultTimeout = 120 * time.Second
)

type Client struct {
	BaseURL    string
	APIKey     string
	SandboxTkn string
	HTTPClient *http.Client
}

func NewClient() *Client {
	baseURL := os.Getenv("SOCKT_API_URL")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	baseURL = strings.TrimRight(baseURL, "/")

	return &Client{
		BaseURL:    baseURL,
		APIKey:     os.Getenv("SOCKT_API_KEY"),
		SandboxTkn: os.Getenv("SOCKT_SANDBOX_TOKEN"),
		HTTPClient: &http.Client{Timeout: DefaultTimeout},
	}
}

func (c *Client) token() string {
	if c.SandboxTkn != "" {
		return c.SandboxTkn
	}
	return c.APIKey
}

func (c *Client) doRequest(method, path string, body interface{}, query map[string]string) ([]byte, int, error) {
	url := c.BaseURL + "/v1" + path

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, fmt.Errorf("marshal request: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("create request: %w", err)
	}

	if tok := c.token(); tok != "" {
		req.Header.Set("Authorization", "Bearer "+tok)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if query != nil {
		q := req.URL.Query()
		for k, v := range query {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()
	}

	var resp *http.Response
	for attempt := 0; attempt < 3; attempt++ {
		resp, err = c.HTTPClient.Do(req)
		if err != nil {
			return nil, 0, fmt.Errorf("request failed: %w", err)
		}
		if resp.StatusCode != http.StatusServiceUnavailable {
			break
		}
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if strings.Contains(string(respBody), "pod_starting") || strings.Contains(string(respBody), "starting") {
			time.Sleep(2 * time.Second)
			continue
		}
		return respBody, resp.StatusCode, nil
	}

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return respBody, resp.StatusCode, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	return respBody, resp.StatusCode, nil
}

func (c *Client) Get(path string, query map[string]string) ([]byte, error) {
	body, _, err := c.doRequest(http.MethodGet, path, nil, query)
	return body, err
}

func (c *Client) Post(path string, payload interface{}) ([]byte, error) {
	body, _, err := c.doRequest(http.MethodPost, path, payload, nil)
	return body, err
}

func (c *Client) Delete(path string) ([]byte, error) {
	body, _, err := c.doRequest(http.MethodDelete, path, nil, nil)
	return body, err
}
