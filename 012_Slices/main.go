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

	// Now When we declare the size and initialized or even not initialize the value then it show Value 0 0 0 0 not NIL.

	var num = make([]int, 10)
	// Output: [0 0 0 0 0 0 0 0 0 0]
	fmt.Println(num)

	// NOTE:  in this make([]int, 2 or 10)
	// Just Show the limit of this slice, it doesn't mean more than 2 elements not store. SLice can store more then 2 element in this num slice.

	// How to check capacity on slice
	fmt.Println(cap(num))
	// Output: 10

	// So in SLICE make() function it have three things

	// var num = make(int[], 2, 10)

	// inside make() function 3 things
	// 1. int[] = inform about the dataType in SLICE.
	// 2. 2 = inform about the LIMIT of the SLICE.
	// 3. 10 = inform about the CAPACITY of the SLICE.

}
