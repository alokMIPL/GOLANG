package main

import (
	"fmt"
	"io"
	"net/http"
)

func main() {

	url := "http://jsonplaceholder.typicode.com/todos"

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println(err)
		return
	}

	defer resp.Body.Close()

	// Here we check the status code, if status code is OK or 200 then proceide otherwise give the error with that status code.
	if resp.StatusCode != http.StatusOK {
		fmt.Println(resp.Status)
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	bodyText := string(bodyBytes)

	fmt.Println(bodyText)

}
