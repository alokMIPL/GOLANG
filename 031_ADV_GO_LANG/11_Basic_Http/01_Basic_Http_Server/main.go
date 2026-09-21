package main

import (
	"fmt"
	"net/http"
)

func helloHandler(w http.ResponseWriter, r *http.Request) {

	// Restricting HTTP Methods. This checks that the request must matched with the given required.
	// For GET accept only GET
	// For POST accept only POST
	// For PATCH accept only PATCH
	if r.Method != http.MethodGet {
		http.Error(w, "only Get is allowed", http.StatusMethodNotAllowed)
		return
	}

	// This return a response for this request.
	_, _ = w.Write([]byte("Hello from GO net/http server"))

}

func main() {

	// http
	// How to register a router.
	http.HandleFunc("/hello", helloHandler)

	fmt.Println("Try going to 8000 port")

	err := http.ListenAndServe(":8080", nil)

	fmt.Println("Server error:", err)

}
