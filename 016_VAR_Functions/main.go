package varfunctions

import "fmt"

// ==========================================
// 1. Basic Variadic Function (Sum)
// ==========================================
func sum(numbers ...int) {
	fmt.Println("Numbers:", numbers)

	total := 0
	for _, num := range numbers {
		total += num
	}

	fmt.Println("Sum:", total)
	fmt.Println()
}

// ==========================================
// 2. Variadic Function with Strings
// ==========================================
func printNames(names ...string) {
	fmt.Println("Names:")

	for i, name := range names {
		fmt.Printf("%d. %s\n", i+1, name)
	}

	fmt.Println()
}

// ==========================================
// 3. Find Maximum Number
// ==========================================
func max(nums ...int) int {

	if len(nums) == 0 {
		return 0
	}

	maximum := nums[0]

	for _, num := range nums {
		if num > maximum {
			maximum = num
		}
	}

	return maximum
}

// ==========================================
// 4. Normal Parameter + Variadic Parameter
// Variadic parameter must always be LAST
// ==========================================
func greet(message string, names ...string) {

	for _, name := range names {
		fmt.Println(message, name)
	}

	fmt.Println()
}

// ==========================================
// Main Function
// ==========================================
func main() {

	// ------------------------------------------------
	// Example 1 : Basic Variadic Function
	// ------------------------------------------------
	fmt.Println("========== Example 1 : Sum ==========")

	sum(10, 20)
	sum(1, 2, 3, 4, 5)
	sum()

	// ------------------------------------------------
	// Example 2 : Print Names
	// ------------------------------------------------
	fmt.Println("========== Example 2 : Names ==========")

	printNames("Alok")
	printNames("Rahul", "Amit", "Priya")

	// ------------------------------------------------
	// Example 3 : Maximum Number
	// ------------------------------------------------
	fmt.Println("========== Example 3 : Maximum ==========")

	fmt.Println("Maximum Number:", max(10, 50, 20, 90, 15))
	fmt.Println()

	// ------------------------------------------------
	// Example 4 : Normal + Variadic Parameter
	// ------------------------------------------------
	fmt.Println("========== Example 4 : Greeting ==========")

	greet("Hello", "Alok", "Rahul", "Amit")

	// ------------------------------------------------
	// Example 5 : Passing Slice to Variadic Function
	// ------------------------------------------------
	fmt.Println("========== Example 5 : Slice ==========")

	numbers := []int{10, 20, 30, 40, 50}

	// Wrong (Compile Error)
	// sum(numbers)

	// Correct
	sum(numbers...)

	// ------------------------------------------------
	// Example 6 : fmt.Println() is Variadic
	// ------------------------------------------------
	fmt.Println("========== Example 6 : fmt.Println ==========")

	fmt.Println("Hello")
	fmt.Println("Hello", "World")
	fmt.Println(10, 20, 30, 40, 50)

	fmt.Println()

	// ------------------------------------------------
	// Example 7 : Checking the Type
	// ------------------------------------------------
	fmt.Println("========== Example 7 : Type ==========")

	checkType(1, 2, 3, 4, 5)
}

// ==========================================
// Variadic Parameter is actually a Slice
// ==========================================
func checkType(nums ...int) {

	fmt.Printf("Type of nums : %T\n", nums)
	fmt.Println("Length :", len(nums))
	fmt.Println("Values :", nums)
}
