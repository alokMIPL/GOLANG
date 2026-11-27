package main

import (
	"encoding/json"
	"fmt"
)

type Person struct {
	Name    string `json:"name"`
	Age     int    `json:"age"`
	IsAdult bool   `json:"isAdult"`
}

func main() {
	person := Person{
		Name:    "John Doe",
		Age:     30,
		IsAdult: true,
	}

	fmt.Println("Direct Structure Data", person)

	// Convert person JSON Encoding (Marshal)
	// Means converting struct into JSON data

	jsonData, err := json.Marshal(person)
	if err != nil {
		fmt.Println("Error:", err)
		return

	}
	fmt.Println("Person Data after marshaling is : ", string(jsonData))

	// converting JSON Decoding (Unmarshal)
	// Means converting JSON data into struct

	var personData Person
	err = json.Unmarshal(jsonData, &personData)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println("Unmarshaled Person Data is : ", personData)
}

// So Direct Structure Data is :  {John Doe 30 true}
// Person Data after marshaling is :  {"name":"John Doe","age":30,"isAdult":true}
// Unmarshaled Person Data is :  {John Doe 30 true}
