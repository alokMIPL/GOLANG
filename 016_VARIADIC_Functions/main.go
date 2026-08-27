package main

import "fmt"

// how to make a variadic function
// Now this sum() function only takes int type
func sum(nums ...int) int {
	total := 0

	for _, num := range nums {
		total = total + num
	}
	return total
}

// If we want to make a function that take any Data Type then we need to use ...any or ...interface.

func anyType(nums ...any) int {
	fmt.Println(nums...)
	return len(nums)
}

func main() {

	// Variadic Function in fmt.Println()
	fmt.Println(1, 2, 3, 4, 5, 6, 7, "Hello")
	// In Variadic function we can pass any number of parameters, basically there is no limit for that.

	result := sum(1, 2, 3, 4, 5, 6, 7, 8)
	fmt.Println(result)

	finalResult := anyType(1, 3, 5, 6, "kola", 345, true, false, "alok")
	fmt.Println(finalResult)

}
