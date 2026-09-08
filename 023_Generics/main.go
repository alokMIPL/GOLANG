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

// ****************
// See in both func printSlice and printStringSlice, we are doing the same thing, but the only difference is the type of slice. So, we can use generics to avoid this duplication of code.

func printGenericSlice[T any](items []T) {
	// In place of any we can use interface{} also, but any is more readable and easy to understand.
	for _, item := range items {
		fmt.Println(item)
	}
}

// Now i want to scope the type so that i can use only int and string in the generic function, so i will create a type constraint for that.

func printGenericScopedSlice[T int | string](items []T) {
	// We can use int | string to scope the type so that we can use only int and string in the generic function.
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

	// Now Print the string Slice using generics
	fmt.Println("*************")
	printGenericSlice(nums)
	fmt.Println("*************")
	// Now Print the string Slice using generics
	printGenericSlice(names)

	// Now Print the string Slice using generics
	fmt.Println("*************")
	printGenericSlice(nums)
	fmt.Println("*************")
	// Now Print the string Slice using generics
	printGenericSlice(names)

}
