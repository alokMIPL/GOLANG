package main

import "fmt"

// how to make a variadic function
func sum(nums ...int) int {
	total := 0

	for _, num := range nums {
		total = total + num
	}
	return total
}

func main() {

	// Variadic Function in fmt.Println()
	fmt.Println(1, 2, 3, 4, 5, 6, 7, "Hello")
	// In Variadic function we can pass any number of parameters, basically there is no limit for that.

	result := sum(1, 2, 43, 5, 6, 7, 8)
	fmt.Println(result)

}
