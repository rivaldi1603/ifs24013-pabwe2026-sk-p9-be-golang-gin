package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// SearchClient defines the contract for querying the live Search Engine service.
type SearchClient interface {
	Search(ctx context.Context, query string) ([]SearchSource, error)
}

type httpSearchClient struct {
	endpoint   string
	httpClient *http.Client
}

// NewSearchClient creates an HTTP SearchClient targeting the configured endpoint.
func NewSearchClient(endpoint string) SearchClient {
	return &httpSearchClient{
		endpoint:   strings.TrimRight(endpoint, "/"),
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *httpSearchClient) Search(ctx context.Context, query string) ([]SearchSource, error) {
	if c.endpoint == "" {
		return nil, fmt.Errorf("search engine endpoint not configured")
	}
	reqURL := fmt.Sprintf("%s?q=%s", c.endpoint, url.QueryEscape(query))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("search engine returned status %d", resp.StatusCode)
	}

	var parsed struct {
		Results []SearchSource `json:"results"`
		Data    []SearchSource `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, err
	}
	if len(parsed.Results) > 0 {
		return parsed.Results, nil
	}
	return parsed.Data, nil
}
