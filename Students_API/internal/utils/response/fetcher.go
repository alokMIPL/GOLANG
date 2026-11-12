package response

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	// DefaultCatFactURL is the public upstream used when no Fetcher is given.
	DefaultCatFactURL = "https://catfact.ninja/fact"

	maxUpstreamBody = 64 << 10 // never read more than 64 KiB from upstream
	maxErrorDrain   = 4 << 10  // drain a little of error bodies for connection reuse
	defaultClientTO = 5 * time.Second
)

var ErrUpstreamStatus = errors.New("unexpected upstream status")

type Fetcher func(ctx context.Context) (*CatFactResponse, error)

func NewHTTPFetcher(client *http.Client, url string) Fetcher {
	if client == nil {
		client = &http.Client{Timeout: defaultClientTO}
	}
	return func(ctx context.Context) (*CatFactResponse, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, fmt.Errorf("build upstream request: %w", err)
		}
		req.Header.Set("Accept", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("call upstream: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxErrorDrain))
			return nil, fmt.Errorf("%w: %s", ErrUpstreamStatus, resp.Status)
		}

		var out CatFactResponse
		if err := json.NewDecoder(io.LimitReader(resp.Body, maxUpstreamBody)).Decode(&out); err != nil {
			return nil, fmt.Errorf("decode upstream body: %w", err)
		}
		return &out, nil
	}
}
