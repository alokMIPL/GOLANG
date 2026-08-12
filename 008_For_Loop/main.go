package main

import "fmt"

// For Loop
// For -> Only construct in GO for Looping
func main() {

	// While loop
	i := 1
	for i <= 3 {
		fmt.Println(i)
		i = i + 1
	}

	// Output : 1 2 3

	// Creating infinite loop
	for {
		fmt.Println(1)
	}

}
