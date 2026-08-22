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
	// 3. 10 = inform about the INITAIL CAPACITY of the SLICE.
	// Link this

	var kk = make([]int, 5, 10)
	fmt.Println(kk)
	// Here we use the CAPACITY as 10 but if we insert the element more than capacity then it resize it.

	// Now how to add element in SLICE.

	// So we hve a function name append()

	var name = make([]int, 2, 5)
	name = append(name, 1)
	name = append(name, 2)
	fmt.Println(name)
	fmt.Println(cap(name))

	// Output: [0 0 1 2]
	// 5

	// Now we insert more than 5 elements
	name = append(name, 3)
	name = append(name, 4)
	name = append(name, 5)
	name = append(name, 6)
	fmt.Println(name)
	fmt.Println(cap(name))
	// Output: [0 0 1 2 3 4 5 6]
	// 10

	// Now see the capacity of the SLICE is doubled from 5 ==>> 10.

	// Now we insert more than 10 elements
	name = append(name, 7)
	name = append(name, 8)
	name = append(name, 9)
	name = append(name, 10)
	name = append(name, 11)
	name = append(name, 12)
	name = append(name, 13)
	fmt.Println(name)
	fmt.Println(cap(name))
	// Output: [0 0 1 2 3 4 5 6 7 8 9 10 11 12 13]
	// 20

	// Now see the capacity of the SLICE is doubled from 10 ==>> 20.

	// Now you notice in every slice output there is 0 0 in first and second place.

	// It comes because we initialize the slice by limit 0 so for the first time when slice execute it fll the sapce by 0 0.

	// So, to get empty SLICE we use limit as 0 like this

}
