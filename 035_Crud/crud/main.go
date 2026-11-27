package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type Todo struct {
	UserID    int    `json:"userId"`
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func main() {
	url := "http://jsonplaceholder.typicode.com/todos"
	url1 := "http://jsonplaceholder.typicode.com/todos/1"
	url2 := "http://jsonplaceholder.typicode.com/todos/1"

	res, err := http.Get(url)
	resp, err := http.Get(url1)
	resp2, err := http.Get(url2)

	if err != nil {
		fmt.Println("Error getting : ", err)
		return
	}

	defer res.Body.Close()
	defer resp.Body.Close()
	defer resp2.Body.Close()

	// We can also check the status code and status message of the response to see if the request was successful or not. The status code is a three-digit number that indicates the outcome of the request, while the status message is a brief description of the status code.

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error in getting response: ", resp.Status)
		return
	}

	// **************
	// Basic method to get data from URL
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body: ", err)
		return
	}

	fmt.Println("Data : ", string(data))

	fmt.Println("Status code", resp.StatusCode)
	fmt.Println("Status code", resp.Status)

	// **************
	// Advanced method to get data from URL
	var todo Todo
	err = json.NewDecoder(resp2.Body).Decode(&todo)
	if err != nil {
		fmt.Println("Error decoding JSON: ", err)
		return
	}

	fmt.Println("Decoded Todo: ", todo)

	// **************
	// We can not write string(todo) because todo is a struct and we can not convert struct into string directly. We need to convert it into JSON bytes first and then we can convert it into string.

	todoBytes, err := json.Marshal(todo)
	if err != nil {
		fmt.Println("Error marshaling JSON: ", err)
		return
	}
	fmt.Println("Decoded Todo: ", string(todoBytes))
}

/* Output
Status code 200
Status code 200 OK
*/
