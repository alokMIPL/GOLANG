package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func successHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	res := map[string]any{
		"ok":       true,
		"message":  "Json encode succesfully",
		"datetime": time.Now().UTC(),
	}

	// encode is writing the JSON and returning back
	_ = json.NewEncoder(w).Encode(res)
}

// Decode is read the request form request body

func main() {

	http.HandleFunc("/ok", successHandler)

	err := http.ListenAndServe(":5000", nil)
	fmt.Println(err)

}
