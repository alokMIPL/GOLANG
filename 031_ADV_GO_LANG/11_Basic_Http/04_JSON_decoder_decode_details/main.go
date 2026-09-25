package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

func writeJSON(w http.ResponseWriter, status int, data any) {

	/*
	   The response we send back (w) needs a "Content-Type" header, so the
	   client knows the body is JSON and can parse it correctly.

	   To set that header, we first need the header collection of the
	   response, which we get using w.Header(). That returned object has
	   a .Set(key, value) method on it, which we use to actually set the
	   Content-Type header.

	   So the two steps combined into one line:
	       w.Header().Set("Content-Type", "application/json")
	*/

	w.Header().Set("Content-Type", "application/json")

	/* `w.WriteHeader(status)`
	This does two things at once, and both matter:
	1. It sets the HTTP status code
	2. It sends/flushes everything staged so far

	w.Header().Set("Content-Type", "application/json")   // staged, not sent yet
	w.WriteHeader(status)                                  // ← NOW it's actually sent
	*/
	w.WriteHeader(status)
	/*
		1. json.NewEncoder(w)
		This creates a new Encoder object, plumbed to write into w (your response). Just like NewDecoder didn't read anything by itself, NewEncoder doesn't write anything by itself yet — it just sets up the tool, connected to its destination (w, in this case, instead of r.Body).

		2. .Encode(data)
		This is where the actual work happens: it takes data (your map, e.g. {"ok": "false", "error": "..."}), converts it into JSON text, and writes that JSON directly into w — meaning, it streams out to the client as the response body.

		3. _=
		_ is Go's blank identifier. It means: "a value exists here, but I don't want it — throw it away."

	*/

	_ = json.NewEncoder(w).Encode(data)
}

type TestRequest struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
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
	// This `var req TestRequest` provide a container to the r.Body data that comes from post method by clinet.
	// And it arrange it inside a type struct

	dec := json.NewDecoder(r.Body)
	// now we create a variable `dec` that have a `json.NewDecoder(r.Body)`
	// the fucntion of `json.NewDecoder()` is only job is to wrap r.Body inside a new Decoder.
	// and then stored the NewDecoder in dec variable.

	// Now in next line `dec.Decode(&req)` this actually decode the (r.Body)
	// is the method that does the real work: pulls bytes from r.Body, parses them as JSON.

	/* **** REMEMBER So: NewDecoder() function = "set up the tool." Decode() = "actually use the tool to do the parsing." Two separate steps, two separate lines.

	dec := json.NewDecoder(r.Body)   // dec = the tool/machine that CAN decode
	dec.Decode(&req)                 // req = where the DECODED RESULT actually ends up

	*/

	// Rought way to handle decode value

	err := dec.Decode(&req)

	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    "false",
			"error": "Json format not correct",
		})
		return
	}
	// Optmized way to handle decode value
	// if err := dec.Decode(&req); err != nil {
	// 	writeJSON(w, http.StatusBadRequest, map[string]any{
	// 		"ok":    "false",
	// 		"error": "Invalid json format",
	// 	})
	// 	return
	// }

	// Just to remove the extra space from the name string.
	req.Name = strings.TrimSpace(req.Name)

	// Now validation check
	if req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    "false",
			"error": "Name must not be empty",
		})
		return
	}
	if req.Age <= 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{
			"ok":    "false",
			"error": "Age must be required",
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
