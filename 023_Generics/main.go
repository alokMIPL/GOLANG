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

/*
comparable is an interface that is implemented by all comparable types (booleans, numbers, strings, pointers, channels, arrays of comparable types, structs whose fields are all comparable types). The comparable interface may only be used as a type parameter constraint, not as the type of a variable.
*/
func printGenericComparableSlice[T comparable](items []T) {
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

// Now Stack
type stack struct {
	elements []int
}

// Now i want to use this stack for string also, so i will create a generic stack.
type genericStack[T any] struct {
	elements []T
}

func main() {

	// It Print the int Slice

	fmt.Println("Printing the int Slice")

	nums := []int{1, 2, 3}
	printSlice(nums)

	fmt.Println("*************")

	// Now Print the string Slice
	names := []string{"GOLANG", "JS", "SQL"}
	fmt.Println("Printing the string Slice")
	printStringSlice(names)

	// Now Print the string Slice using generics
	fmt.Println("*************")
	printGenericSlice(nums)
	fmt.Println("Printing the int Slice by using generics")
	fmt.Println("*************")
	// Now Print the string Slice using generics
	printGenericSlice(names)
	fmt.Println("Printing the string Slice by using generics")

	// Now Print the string Slice using Scope generics
	fmt.Println("*************")
	printGenericScopedSlice(nums)
	fmt.Println("Printing the int Slice by using Scope generics")
	fmt.Println("*************")
	// Now Print the string Slice using Scope generics
	printGenericScopedSlice(names)
	fmt.Println("Printing the string Slice by using Scope generics")

	// In Scope generics we can use only int and string, if we try to use any other type then it will give an error.

	// decimalNumber := []float32{1.9, 2.5, 3.3}
	// printGenericScopedSlice(decimalNumber)

	// Same for other data Types like float64, bool, etc. we can not use them in the scope generics function, beacuse we only declare INT and STRING in the scope generics function. So, if we try to use any other type then it will give an error.

	printGenericComparableSlice(nums)
	fmt.Println("Printing the int Slice by using comparable generics")
	fmt.Println("*************")
	printGenericComparableSlice(names)
	fmt.Println("Printing the string Slice by using comparable generics")

	fmt.Println("*************")

	fmt.Println("Now Stack")

	// Now Stack
	myStack := stack{
		elements: []int{1, 2, 3},
	}
	fmt.Println("Basic Stack value", myStack)

	// Now Generic Stack
	fmt.Println("*************")

	fmt.Println("Now Stack")
	myGenericStack := genericStack[string]{
		elements: []string{"GOLANG", "JS", "SQL"},
	}

	fmt.Println("Generic Stack value", myGenericStack)

}
