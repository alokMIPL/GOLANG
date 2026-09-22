package main

import (
	"fmt"
	"net/http"
)

func main() {

	err := http.ListenAndServe(":5000", nil)
	fmt.Println(err)

}
