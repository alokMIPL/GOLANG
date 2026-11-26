package main

import "fmt"

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

	fmt.Println(person)
}
