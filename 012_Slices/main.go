package main

import "fmt"

// Slices
// It is basically Dynamic Array
// In slice we don't give the length of the slice. That's why we called it Dynamic Array.
// most used construct in GO
// + useful methods

func main() {

	// uninitialized slice is nil not empty
	var nums []int
	fmt.Println(nums)
	// Output: [] nil not empty

	// Check it length
	fmt.Println(len(nums))
	// Output: 0

	// Check it is nil or not
	fmt.Println(nums == nil)
	// Output: true

	// To avoide NIL in slice we use "make()" function
	// here in make() function we take our slice var as integer and size of slice is 2
	// make([]int, 10)

	// var num = make([]int, 10)
	// Output: [0 0]

	var num = make([]int, 10)
	// Output: [0 0 0 0 0 0 0 0 0 0]
	fmt.Println(num)

}
