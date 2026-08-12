package main

import "fmt"

// Note
// We can declare the variable outside the func main() function

// Ex
var age int = 30

const className string = "Golang"

// Note Note
// We can't use variable shortMethod outside the func main() function

// For ex
// carName := "Honda"
// Get this error "expected declaration, found carNamesyntax"

func main() {

	fmt.Println("Age is : = ", age, "Class Name is : = ", className)

	// Constants

	const name = "goLang"

	// Can't Redecalre the constant values
	// name = "Script"
	/*
		cannot assign to name (neither addressable nor a map index expression)compilerUnassignableOperand
		const name untyped string = "goLang"
	*/

	// We can declare and initialize Multiple Constant same time.
	const (
		port = 5000
		host = "localhost"
	)

	fmt.Println(port, host)

}
