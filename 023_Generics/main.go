package main

import "fmt"

func printSlice(items []int) {
	for _, item := range items {
		fmt.Println(item)
	}
}

func printStringSlice(items []string) {
	for _, item := range items {
		fmt.Println(item)
	}
}

func main() {

	// It Print the int Slice
	nums := []int{1, 2, 3}
	printSlice(nums)

	fmt.Println("*************")

	// Now Print the string Slice
	names := []string{"GOLANG", "JS", "SQL"}
	printStringSlice(names)
}
