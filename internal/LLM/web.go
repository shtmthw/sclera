package llm

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

// searxngBaseURL points at the local SearXNG instance from your
// ~/services/searxng docker-compose setup. Move this to config/env if you
// ever deploy this somewhere other than localhost.
const searxngBaseURL = "http://localhost:8080"

// searxHTTPClient is package-level so it can reuse connections across calls
// instead of creating a new client (and TLS/pool setup) every search.
var searxHTTPClient = &http.Client{
	Timeout: 15 * time.Second,
}

// searxResult mirrors the fields we care about from a single SearXNG
// /search?format=json result entry.
type searxResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Content string `json:"content"`
}

type searxResponse struct {
	Results []searxResult `json:"results"`
}

// SearchWeb queries the local SearXNG instance and returns a plain-text
// summary of the top results, formatted to be fed back into Gemma as a
// tool result message.
func SearchWeb(ctx context.Context, query string) (string, error) {
	endpoint := searxngBaseURL + "/search?" + url.Values{
		"q":      {query},
		"format": {"json"},
	}.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("building searxng request: %w", err)
	}

	resp, err := searxHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("calling searxng: %w", err)
	}

	defer func() error {
		rerr := resp.Body.Close()
		if rerr != nil {
			return rerr
		}
		return nil
	}()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("searxng returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var parsed searxResponse
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return "", fmt.Errorf("decoding searxng response: %w", err)
	}

	if len(parsed.Results) == 0 {
		return "No results found.", nil
	}

	const maxResults = 5
	if len(parsed.Results) > maxResults {
		parsed.Results = parsed.Results[:maxResults]
	}

	var b strings.Builder
	for i, r := range parsed.Results {
		fmt.Fprintf(&b, "%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Content)
	}

	return b.String(), nil
}
