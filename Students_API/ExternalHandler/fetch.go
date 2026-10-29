package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const (
	maxErrBody  = 4 << 10  // cap on how much of an error response we read
	maxOKBody   = 64 << 10 // cap on how much of a success response we read
	clientLimit = 5 * time.Second
)

// ErrEmptyFact is returned when the API answers 200 but without a usable fact.
var ErrEmptyFact = errors.New("catfact: empty fact in response")

// FactFetcher retrieves a single fact. It is an interface so handlers can be
// tested with a fake instead of hitting the network.
type FactFetcher interface {
	Fetch(ctx context.Context) (string, error)
}

type catFactFetcher struct {
	client *http.Client
	url    string
}

// NewCatFactFetcher returns a FactFetcher for the given URL
// (for example "https://catfact.ninja/fact").
// If client is nil, a client with a 5s timeout is used.
func NewCatFactFetcher(client *http.Client, url string) FactFetcher {
	if client == nil {
		client = &http.Client{Timeout: clientLimit}
	}
	return &catFactFetcher{client: client, url: url}
}

func (f *catFactFetcher) Fetch(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return "", fmt.Errorf("catfact: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("catfact: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read a bounded snippet for diagnostics. Reading it also lets the
		// connection be reused by the transport.
		snippet, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
		return "", fmt.Errorf("catfact: unexpected status %d: %s",
			resp.StatusCode, strings.TrimSpace(string(snippet)))
	}

	var payload struct {
		Fact string `json:"fact"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxOKBody)).Decode(&payload); err != nil {
		return "", fmt.Errorf("catfact: decode response: %w", err)
	}

	fact := strings.TrimSpace(payload.Fact)
	if fact == "" {
		return "", ErrEmptyFact
	}
	return fact, nil
}
