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

func main() {
	result := add(3, 5)
	fmt.Println(result)
	// Output = 8
	fmt.Println(getLanguages())
	// Output = golang javaScript c++

	fmt.Println(getLanguagesMixed())
	// Output = golang javaScript true

}
