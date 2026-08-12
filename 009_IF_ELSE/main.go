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

	fmt.Println("****************")

	// Logical OR in IF_ELSE

	var role = "admin"
	var hasPermissions = false

	// OR operator
	if role == "admin" || hasPermissions {
		fmt.Println("yes")
	}

	fmt.Println("****************")

	// And operator
	if role == "admin" && hasPermissions {
		fmt.Println("yes")
	}

	// We can declare variable inside IF_ELSE
	if number := 78; number >= 80 {
		fmt.Println("Topper", number)
	} else if number >= 60 {
		fmt.Println("Average", number)
	} else {
		fmt.Println("Below Average", number)
	}

}
