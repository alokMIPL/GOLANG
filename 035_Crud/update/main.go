package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Todo struct {
	UserID    int    `json:"userId"`
	ID        int    `json:"id"`
	Title     string `json:"title"`
	Completed bool   `json:"completed"`
}

func performGetRequest() {
	url := "http://jsonplaceholder.typicode.com/todos/1"

	resp, err := http.Get(url)

	if err != nil {
		fmt.Println("Error getting : ", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error in getting response: ", resp.Status)
		return
	}
	// **************
	// Advanced method to get data from URL
	var todo Todo
	err = json.NewDecoder(resp.Body).Decode(&todo)
	if err != nil {
		fmt.Println("Error decoding JSON: ", err)
		return
	}
	fmt.Println("Decoded Todo: ", todo)
}

func performPostRequest() {
	todo := Todo{
		UserID:    22,
		Title:     "New Todo",
		Completed: true,
	}

	// Now want to send this todo to the server using POST request. We need to convert this todo struct into JSON bytes first and then we can send it to the server.
	// Because the server expects the request body to be in JSON format. So we need to convert this todo struct into JSON bytes first and then we can send it to the server.
	// So we use json.Marshal function to convert this todo struct into JSON bytes.
	jsonData, err := json.Marshal(todo)
	if err != nil {
		fmt.Println("Error in converting Struct Data to JSON", err)
		return
	}
	fmt.Println("JSON Data after conversion from struct to JSON : ", string(jsonData))
	jsonString := string(jsonData)

	// Now for HTTP.POST we need to convert the string data to reader
	jsonReader := strings.NewReader(jsonString)

	// Now HTTP POST
	// It required three parameters
	// 1. URL
	// 2. Content-Type
	// 3. Request Body
	// So we can use http.Post() function to send this JSON data to the server using POST request.
	// The http.Post() function returns two values
	// 1. Response
	// 2. Error

	myURL := "http://jsonplaceholder.typicode.com/todos"

	resp, err := http.Post(myURL, "application/json", jsonReader)
	if err != nil {
		fmt.Println("Error in POST request: ", err)
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		fmt.Println("Error in getting response StatusCode Error: ", resp.Status)
		return
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body: ", err)
		return
	}

	fmt.Println("Response: ", string(data))
	fmt.Println("Response status", resp.Status)
}

func performUpdateRequest() {
	todo := Todo{
		UserID:    23768,
		Title:     "New Todo Update",
		Completed: true,
	}

	jsonData, err := json.Marshal(todo)
	if err != nil {
		fmt.Println("Error in converting Struct Data to JSON", err)
		return
	}
	fmt.Println("JSON Data after conversion from struct to JSON : ", string(jsonData))
	jsonString := string(jsonData)

	jsonReader := strings.NewReader(jsonString)

	myPostURL := "http://jsonplaceholder.typicode.com/todos/1"

	// Create PUT Request

	req, err := http.NewRequest(http.MethodPut, myPostURL, jsonReader)
	if err != nil {
		fmt.Println()
	}

}

func main() {
	performGetRequest()
	performPostRequest()
	performUpdateRequest()
}

/*
http.StatusOK	== 200	Request succeeded; no new resource was created.
http.StatusCreated ==	201	Request succeeded; a new resource was created.
*/
