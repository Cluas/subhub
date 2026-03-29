package engine

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"sigs.k8s.io/yaml"
)

const maxResponseSize = 10 * 1024 * 1024 // 10MB

// httpClient is a globally shared HTTP client.
var httpClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	},
}

// FetchClashConfig fetches URL content and converts it to JSON (supports both YAML and JSON formats).
func FetchClashConfig(ctx context.Context, subscribeURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, subscribeURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "ClashMetaForAndroid/2.0 Mihomo/1.0 subhub/1.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch url: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		slog.Debug("not valid YAML, trying as raw JSON", "url", subscribeURL)
		return data, nil
	}
	return jsonData, nil
}
