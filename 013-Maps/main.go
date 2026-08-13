package main

import "fmt"

// Struct for map example
type Student struct {
	Name string
	Age  int
}

func main() {

	// =====================================================
	// 1. Creating a Map using make()
	// =====================================================

	fmt.Println("===== Creating Map =====")

	ages := make(map[string]int)

	fmt.Println("Empty Map:", ages)

	// =====================================================
	// 2. Adding Elements
	// =====================================================

	ages["Alice"] = 22
	ages["Bob"] = 25
	ages["Charlie"] = 28

	fmt.Println("\nAfter Adding Values:")
	fmt.Println(ages)

	// =====================================================
	// 3. Accessing Values
	// =====================================================

	fmt.Println("\n===== Accessing Values =====")

	fmt.Println("Alice Age:", ages["Alice"])
	fmt.Println("Bob Age:", ages["Bob"])

	// Key doesn't exist
	fmt.Println("David Age:", ages["David"]) // Returns 0

	// =====================================================
	// 4. Checking if Key Exists
	// =====================================================

	fmt.Println("\n===== Checking Key Exists =====")

	age, found := ages["Bob"]

	if found {
		fmt.Println("Bob exists with age:", age)
	} else {
		fmt.Println("Bob not found")
	}

	age, found = ages["David"]

	if found {
		fmt.Println("David exists with age:", age)
	} else {
		fmt.Println("David not found")
	}

	// =====================================================
	// 5. Updating Value
	// =====================================================

	fmt.Println("\n===== Updating Value =====")

	ages["Alice"] = 30

	fmt.Println("Updated Map:")
	fmt.Println(ages)

	// =====================================================
	// 6. Deleting Key
	// =====================================================

	fmt.Println("\n===== Deleting Key =====")

	delete(ages, "Charlie")

	fmt.Println(ages)

	// =====================================================
	// 7. Length of Map
	// =====================================================

	fmt.Println("\n===== Length =====")

	fmt.Println("Length:", len(ages))

	// =====================================================
	// 8. Iterating over Map
	// =====================================================

	fmt.Println("\n===== Iterating =====")

	for key, value := range ages {
		fmt.Println(key, "->", value)
	}

	// =====================================================
	// 9. Map Literal
	// =====================================================

	fmt.Println("\n===== Map Literal =====")

	countries := map[string]string{
		"IN": "India",
		"US": "United States",
		"JP": "Japan",
	}

	fmt.Println(countries)

	// =====================================================
	// 10. Nested Map
	// =====================================================

	fmt.Println("\n===== Nested Map =====")

	students := map[string]map[string]int{
		"Alice": {
			"Math":    95,
			"Science": 90,
		},
		"Bob": {
			"Math":    85,
			"Science": 88,
		},
	}

	fmt.Println("Alice Math:", students["Alice"]["Math"])
	fmt.Println("Bob Science:", students["Bob"]["Science"])

	// =====================================================
	// 11. Map with Struct
	// =====================================================

	fmt.Println("\n===== Map with Struct =====")

	studentMap := map[int]Student{
		1: {
			Name: "Rahul",
			Age:  20,
		},
		2: {
			Name: "Amit",
			Age:  22,
		},
	}

	fmt.Println(studentMap)

	fmt.Println("Student ID 1:", studentMap[1])

	// =====================================================
	// 12. Nil Map
	// =====================================================

	fmt.Println("\n===== Nil Map =====")

	var marks map[string]int

	fmt.Println(marks) // nil

	fmt.Println("Reading Nil Map:", marks["Math"])

	// Uncommenting below line will cause panic
	// marks["Math"] = 100

	// =====================================================
	// 13. Map as Frequency Counter
	// =====================================================

	fmt.Println("\n===== Frequency Counter =====")

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

	fmt.Println(frequency)

	// =====================================================
	// 14. Using Map as Set
	// =====================================================

	fmt.Println("\n===== Map as Set =====")

	set := make(map[string]bool)

	set["Apple"] = true
	set["Banana"] = true
	set["Orange"] = true

	fmt.Println(set)

	if set["Apple"] {
		fmt.Println("Apple exists")
	}

	if set["Mango"] {
		fmt.Println("Mango exists")
	} else {
		fmt.Println("Mango not found")
	}

	// =====================================================
	// 15. Clearing Map
	// =====================================================

	fmt.Println("\n===== Clearing Map =====")

	clear(ages)

	fmt.Println(ages)
	fmt.Println("Length:", len(ages))
}
