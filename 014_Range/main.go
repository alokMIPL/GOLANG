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

	fmt.Println("***********")

	// Now to print number with INDEX so _, is replaced by i and i is INDEX in RANGE.

	// Now to access SLICE by RANGE
	for i, num := range nums {
		fmt.Println(num, i)
	}

	/* Output =
	6 0
	7 1
	8 2
	*/

	fmt.Println("***********")

	// Now access MAP by using RANGE

	m := map[string]string{"fname": "John", "lname": "doe"}

	for k, v := range m {
		fmt.Println(k, v)
	}

	fmt.Println("***********")

	// Now access String by using RANGE
	// Basically give UNICODE

	for i, c := range "golang" {
		fmt.Println(i, c)
	}

	/* Output =
	0 103
	1 111
	2 108
	3 97
	4 110
	5 103
	*/

	// Now getting each and every charater of STRING then use string() function.

	for i, c := range "golang" {
		fmt.Println(i, string(c))
	}

	/*
		0 g
		1 o
		2 l
		3 a
		4 n
		5 g
	*/

}
