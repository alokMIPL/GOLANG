package main

import "fmt"

// Struct used in one of the examples
type Student struct {
	Name string
	Age  int
}

func main() {

	// =====================================================
	// 1. Range with Array
	// =====================================================

	fmt.Println("========== RANGE WITH ARRAY ==========")

	arr := [5]int{10, 20, 30, 40, 50}

	for index, value := range arr {
		fmt.Println("Index:", index, "Value:", value)
	}

	// =====================================================
	// 2. Range with Slice
	// =====================================================

	fmt.Println("\n========== RANGE WITH SLICE ==========")

	numbers := []int{100, 200, 300, 400, 500}

	for index, value := range numbers {
		fmt.Println("Index:", index, "Value:", value)
	}

	// =====================================================
	// 3. Ignore Index
	// =====================================================

	fmt.Println("\n========== IGNORE INDEX ==========")

	for _, value := range numbers {
		fmt.Println(value)
	}

	// =====================================================
	// 4. Ignore Value
	// =====================================================

	fmt.Println("\n========== IGNORE VALUE ==========")

	for index := range numbers {
		fmt.Println(index)
	}

	// =====================================================
	// 5. Range with String
	// =====================================================

	fmt.Println("\n========== RANGE WITH STRING ==========")

	name := "Golang"

	for index, character := range name {
		fmt.Printf("Index: %d Character: %c\n", index, character)
	}

	// =====================================================
	// 6. Unicode String
	// =====================================================

	fmt.Println("\n========== UNICODE STRING ==========")

	word := "नमस्ते"

	for index, character := range word {
		fmt.Printf("Index: %d Character: %c\n", index, character)
	}

	// =====================================================
	// 7. Range with Map
	// =====================================================

	fmt.Println("\n========== RANGE WITH MAP ==========")

	students := map[string]int{
		"Alice": 90,
		"Bob":   85,
		"John":  95,
	}

	for key, value := range students {
		fmt.Println(key, "->", value)
	}

	// =====================================================
	// 8. Map Keys Only
	// =====================================================

	fmt.Println("\n========== MAP KEYS ==========")

	for key := range students {
		fmt.Println(key)
	}

	// =====================================================
	// 9. Map Values Only
	// =====================================================

	fmt.Println("\n========== MAP VALUES ==========")

	for _, value := range students {
		fmt.Println(value)
	}

	// =====================================================
	// 10. Range with Struct Slice
	// =====================================================

	fmt.Println("\n========== STRUCT SLICE ==========")

	studentList := []Student{
		{Name: "Rahul", Age: 20},
		{Name: "Amit", Age: 22},
		{Name: "Priya", Age: 19},
	}

	for index, student := range studentList {
		fmt.Println(index, student.Name, student.Age)
	}

	// =====================================================
	// 11. Sum of Slice
	// =====================================================

	fmt.Println("\n========== SUM ==========")

	sum := 0

	for _, value := range numbers {
		sum += value
	}

	fmt.Println("Sum =", sum)

	// =====================================================
	// 12. Average
	// =====================================================

	fmt.Println("\n========== AVERAGE ==========")

	average := sum / len(numbers)

	fmt.Println("Average =", average)

	// =====================================================
	// 13. Maximum Value
	// =====================================================

	fmt.Println("\n========== MAXIMUM ==========")

	max := numbers[0]

	for _, value := range numbers {

		if value > max {
			max = value
		}

	}

	fmt.Println("Maximum =", max)

	// =====================================================
	// 14. Minimum Value
	// =====================================================

	fmt.Println("\n========== MINIMUM ==========")

	min := numbers[0]

	for _, value := range numbers {

		if value < min {
			min = value
		}

	}

	fmt.Println("Minimum =", min)

	// =====================================================
	// 15. Count Even Numbers
	// =====================================================

	fmt.Println("\n========== EVEN NUMBERS ==========")

	even := 0

	for _, value := range numbers {

		if value%2 == 0 {
			even++
		}

	}

	fmt.Println("Even Count =", even)

	// =====================================================
	// 16. Count Odd Numbers
	// =====================================================

	fmt.Println("\n========== ODD NUMBERS ==========")

	odd := 0

	values := []int{1, 2, 3, 4, 5, 6, 7}

	for _, value := range values {

		if value%2 != 0 {
			odd++
		}

	}

	fmt.Println("Odd Count =", odd)

	// =====================================================
	// 17. Double Slice Elements
	// =====================================================

	fmt.Println("\n========== MODIFY SLICE ==========")

	data := []int{2, 4, 6, 8}

	for index := range data {
		data[index] *= 2
	}

	fmt.Println(data)

	// =====================================================
	// 18. Matrix Traversal
	// =====================================================

	fmt.Println("\n========== MATRIX ==========")

	matrix := [][]int{
		{1, 2, 3},
		{4, 5, 6},
		{7, 8, 9},
	}

	for rowIndex, row := range matrix {

		for columnIndex, value := range row {

			fmt.Printf("matrix[%d][%d] = %d\n",
				rowIndex,
				columnIndex,
				value)

		}

	}

	// =====================================================
	// 19. Frequency Counter
	// =====================================================

	fmt.Println("\n========== FREQUENCY ==========")

	words := []string{
		"go",
		"java",
		"go",
		"python",
		"go",
		"java",
	}

	frequency := make(map[string]int)

	for _, word := range words {
		frequency[word]++
	}

	for key, value := range frequency {
		fmt.Println(key, ":", value)
	}

	// =====================================================
	// 20. Range with Channel
	// =====================================================

	fmt.Println("\n========== CHANNEL ==========")

	ch := make(chan int)

	go func() {

		for i := 1; i <= 5; i++ {
			ch <- i * 10
		}

		close(ch)

	}()

	for value := range ch {
		fmt.Println(value)
	}

	// =====================================================
	// 21. Search Element
	// =====================================================

	fmt.Println("\n========== SEARCH ==========")

	target := 300
	found := false

	for _, value := range numbers {

		if value == target {
			found = true
			break
		}

	}

	if found {
		fmt.Println(target, "Found")
	} else {
		fmt.Println(target, "Not Found")
	}

	// =====================================================
	// 22. Skip using Continue
	// =====================================================

	fmt.Println("\n========== CONTINUE ==========")

	for _, value := range values {

		if value%2 == 0 {
			continue
		}

		fmt.Println(value)

	}

	// =====================================================
	// 23. Break Example
	// =====================================================

	fmt.Println("\n========== BREAK ==========")

	for _, value := range values {

		if value == 5 {
			break
		}

		fmt.Println(value)

	}

	// =====================================================
	// 24. Reverse Traversal (Classic for Loop)
	// =====================================================

	fmt.Println("\n========== REVERSE ==========")

	for i := len(numbers) - 1; i >= 0; i-- {
		fmt.Println(numbers[i])
	}

	// =====================================================
	// 25. Nested Range with Strings
	// =====================================================

	fmt.Println("\n========== WORDS ==========")

	names := []string{
		"Alice",
		"Bob",
		"Charlie",
	}

	for _, name := range names {

		fmt.Println("Word:", name)

		for _, letter := range name {
			fmt.Printf("%c ", letter)
		}

		fmt.Println()

	}

	fmt.Println("\n========== END OF PROGRAM ==========")
}
