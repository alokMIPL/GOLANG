package response

import (
	"net/http"
	"time"
)

const (
	maxErrBody  = 4 << 10
	maxOKBody   = 64 << 10
	clientLimit = 5 * time.Second
)

func NewCatFactFetcher(client *http.Client, url string) FactFetcher {
	if client == nil {
		client = &http.Client(Timeout: clientLimit)
	}
}