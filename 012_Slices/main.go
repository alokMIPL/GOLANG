package main

import "fmt"

// Slices
// It is basically Dynamic Array
// most used construct in GO
// + useful methods

func main() {

	// uninitialized slice is nil not empty
	var nums []int
	fmt.Println(nums)
	// Output: [] nil not empty

	// Check it is nil or not
	fmt.Println(nums == nil)
	// Output: true
}
