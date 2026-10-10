package response

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
	"unicode/utf8"
)

const (
	defaultURL      = "https://catfact.ninja/fact"
	upstreamTimeout = 5 * time.Second
	maxUpstreamBody = 1 << 20 // 1 MiB
)

type CatFactResponse struct {
	Fact   string `json:"fact"`
	Length int    `json:"length"`
}

// NewCatFactFetcher returns a FactFetcher backed by the given client and URL.
// A nil client gets a default one with a timeout as a safety net; the
// per-call deadline normally comes from the context.
func NewCatFactFetcher(client *http.Client, url string) FactFetcher {
	if client == nil {
		client = &http.Client{Timeout: upstreamTimeout}
	}
	return func(ctx context.Context) (*CatFactResponse, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("do request: %w", err)
		}
		defer resp.Body.Close()

		body := io.LimitReader(resp.Body, maxUpstreamBody)
		if resp.StatusCode != http.StatusOK {
			_, _ = io.Copy(io.Discard, body) // let the connection be reused
			return nil, fmt.Errorf("unexpected upstream status %d", resp.StatusCode)
		}

		var fact CatFactResponse
		if err := json.NewDecoder(body).Decode(&fact); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}
		if fact.Fact == "" {
			return nil, fmt.Errorf("upstream returned no fact")
		}
		if fact.Length == 0 {
			fact.Length = utf8.RuneCountInString(fact.Fact)
		}
		return &fact, nil
	}
}
