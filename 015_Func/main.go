package main

import "fmt"

/*
In this line "func add(a int, b int) int {"

this line represent the input taken by function add(a int, b int) and type is int.

but this int { define that there is a return type and must be int.

*/

// Function 1
func add(a int, b int) int {
	// we can also write as
	// add(a, b int)
	return a + b
}

// Function 2
// We can return multiple values in GOLANG Function.
func getLanguages() (string, string, string) {
	return "golang", "javaScript", "c++"
}

// Function 3
// We can return multiple values in GOLANG Function.
func getLanguagesMixed() (string, string, bool) {
	return "golang", "javaScript", true
}

// Basically in GOLANG we retun two things the function value and the secind one is error.

// To compress any compiler error we use "_" to supress that error.

// GOLANG functions are first class citizen functions() and can be used or assigned those functions to any variable and pass those functions as arguments for other functions.

func apply(a, b int, op func(int, int) int) int {
	return op(a, b)
}

func adds(x, y int) int {
	return x + y
}

func multiply(x, y int) int {
	return x * y
}

func main() {
	result := add(3, 5)
	fmt.Println(result)
	// Output = 8
	fmt.Println(getLanguages())
	// Output = golang javaScript c++

	fmt.Println(getLanguagesMixed())
	// Output = golang javaScript true

	// Assign a function to a variable
	greet := func(name string) string {
		return "Hello, " + name
	}

	fmt.Println(greet("Alok"))
	// Output = Hello, Alok

	fmt.Println(apply(3, 4, add))
	fmt.Println(apply(3, 4, multiply))
	// Output = 7 12

}
