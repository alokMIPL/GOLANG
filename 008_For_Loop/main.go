package main

import "fmt"

// For Loop
// For -> Only construct in GO for Looping
func main() {

	// While loop
	i := 1
	for i <= 3 {
		fmt.Println(i)
		i = i + 1
	}

	fmt.Println("***************")

	// Output : 1 2 3

	// Creating infinite loop
	// for {
	// 	fmt.Println(1)
	// }

	fmt.Println("***************")

	// Classic For Loop
	for i := 0; i < 3; i++ {
		fmt.Println(i)
	}
	// Output we get : 	0	 	1	 	2

	fmt.Println("***************")

	// Now Break in For Loop
	// Break stop the loop and come out from loop
	for i := 0; i < 3; i++ {
		if i == 2 {
			break
		}
		fmt.Println(i)
	}
	// Output we get : 	0 	1

	fmt.Println("***************")

	// Now Continue in For loop
	// Continue skip the recent loop and go further.
	for i := 0; i <= 3; i++ {
		if i == 1 {
			continue
		}
		fmt.Println(i)
	}
	// Output we get : 	0 	2 	3

	fmt.Println("***************")

	// New update in GOLANG

	// Range

}
