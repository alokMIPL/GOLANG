package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

type TestRequest struct {
	Name string `json:"name"`
}

func testHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]any{
			"ok":    "false",
			"error": "Only post is allowed",
		})
		return
	}

	defer r.Body.Close()

	var req TestRequest

	dec := json.NewDecoder(r.Body)

	if err := dec.Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    "false",
			"error": "Invalid json format",
		})
		return
	}

	// Now validation check
	req.Name = strings.TrimSpace(req.Name)

	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    "false",
			"error": "Name must not be empty",
		})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"Ok":        "true",
		"data":      req,
		"timeStamp": time.Now().UTC(),
	})

}

func main() {

	http.HandleFunc("/test", testHandler)

	err := http.ListenAndServe(":5000", nil)
	fmt.Println(err)

}

/*

When we test this url `http://localhost:5000/test`

1. If we not pass or post anything then we get this error.
 {
    "error": "Invalid json format",
    "ok": "false"
}

2. If we pass other method then POST then we get this error.
{
    "error": "Only post is allowed",
    "ok": "false"
}

3. If we post empty body then we get this error.
{
    "error": "Name must not be empty",
    "ok": "false"
}

4. If we post evetything correct then we get this Output.

{
    "Ok": "true",
    "data": {
        "name": "alok"
    },
    "timeStamp": "2026-09-25T04:16:35.8374135Z"
}


*/
