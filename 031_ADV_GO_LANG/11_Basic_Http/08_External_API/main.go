package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CatFactResponse struct {
	Fact   string `json:"fact"`
	Length string `json:"length"`
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// This fetchCatFact func take nothing in INPUT but return something
// Now this function return two things CatFactResponse and error
func fetchCatFact() (CatFactResponse, error) {
	url := "https://catfact.ninja/fact"

	// Here the res is doing res — a response object containing the status code, headers, and an open stream.
	res, err := http.Get(url)
	if err != nil {
		return CatFactResponse{}, err
		// when error comes then response in CatFactResponse{} is empty stirng we set, have nothing and only err we get right
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return CatFactResponse{}, fmt.Errorf("external api failed: %s", res.Status)
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		return CatFactResponse{}, err
	}

	var data CatFactResponse
	if err := json.Unmarshal(bodyBytes, &data); err != nil {
		return CatFactResponse{}, err
	}
	return data, nil

}

func externalHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"ok":    "false",
			"error": "Only Get method is allowed.",
		})
		return
	}

	data, err := fetchCatFact()
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    "false",
			"error": "Failed to fetch data.",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":        "true",
		"timeStamp": time.Now().UTC(),
		"external": map[string]any{
			"source": "Catfact.mimja",
			"fact":   data.Fact,
			"length": data.Length,
		},
	})

}

func main() {

	http.HandleFunc("/external", externalHandler)

	err := http.ListenAndServe(":5000", nil)

	fmt.Println(err)

}
