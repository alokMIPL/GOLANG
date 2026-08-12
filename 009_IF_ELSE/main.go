package main

import "fmt"

func main() {

	age := 8

	// if Condition
	if age >= 18 {
		fmt.Println("Person is an Adult")
	} else {
		fmt.Println("Person is not an Adult")
	}

	fmt.Println("****************")

	// else if condition
	if age >= 18 {
		fmt.Println("Person is an Adult")
	} else if age >= 12 {
		fmt.Println("Person is Teenager")
	} else {
		fmt.Println("Person is Kid")
	}

}
