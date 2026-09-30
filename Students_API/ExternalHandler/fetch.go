package response

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const catFactURL = "https://catfact.ninja/fact"

// A shared client with a timeout: the default http.Client has none,
// so a hung upstream would block forever.
var httpClient = &http.Client{Timeout: 5 * time.Second}

func fetchCatFact(ctx context.Context) (*CatFactResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, catFactURL, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call upstream: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}

	var out CatFactResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &out, nil
}
