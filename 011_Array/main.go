package main

import "fmt"

// Array is Number sequence of specific length
func main() {

	// Declaration of Array
	var nums [4]int

	// Finding the lenght of Array
	fmt.Println(len(nums))
	// Output: 4

	// Add element in array at particular INDEX
	nums[0] = 1
	fmt.Println(nums[0])
	// Output: 1

	// Printing the whole array
	fmt.Println(nums)
	// Output: [1 0 0 0]

}
