package main

import "fmt"

// maps => hash, object, dict in different programmign languages.

func main() {

	// Creating map
	m := make(map[string]string)

	// Setting an element
	m["name"] = "golang"
	m["area"] = "backend"

	// Get an element
	fmt.Println(m["name"], m["area"])
	// Output = golang
	// Output = golang backend

}
