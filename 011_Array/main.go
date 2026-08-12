package main

import "fmt"

// Array is Number sequence of specific length
func main() {

	// Declaration of Array
	var nums [4]int

	// Finding the lenght of Array
	fmt.Println(len(nums))
	// Output: 4

	fmt.Println("*****************")

	// Add element in array at particular INDEX
	nums[0] = 1
	fmt.Println(nums[0])
	// Output: 1

	fmt.Println("*****************")

	// Printing the whole array
	fmt.Println(nums)
	// Output: [1 0 0 0]
	// As we see the array have only one index[0] vlaue initialized and rest for index are not initalized by any valuye so, in GOLANG it does not store Garbage value to it. It Store 0 0 0 for index[1], index[2] and index[3]

	fmt.Println("*****************")

	// Same for Boolean also
	var vals [4]bool
	fmt.Println(vals)
	// Output: [false false false false]

	fmt.Println("*****************")

}
