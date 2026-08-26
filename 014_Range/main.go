package main

import "fmt"

// Iterating over data structures
func main() {

	nums := []int{6, 7, 8}

	// printing ARRAY by using FOR loop
	for i := 0; i < len(nums); i++ {
		fmt.Println(nums[i])
	}
	// Output = 6 7 8

	fmt.Println("***********")

	// Sum of Array by using RANGE

	sum := 0

	for _, num := range nums {
		sum = sum + num
		fmt.Println(num)
		// Output = 6 7 8
	}
	fmt.Println(sum)
	// Output = 21

	// Now to print number with INDEX so _, is replaced by i and i is INDEX in RANGE.

	for i, num := range nums {
		fmt.Println(num, i)
	}

	/* Output =
	6 0
	7 1
	8 2

	*/

}
