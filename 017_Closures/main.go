package closure

import "fmt"

// ==========================================
// Example 1: Basic Closure
// ==========================================
func outer() func() {

	message := "Hello Golang"

	return func() {
		fmt.Println(message)
	}
}

// ==========================================
// Example 2: Counter Using Closure
// ==========================================
func counter() func() int {

	count := 0

	return func() int {
		count++
		return count
	}
}

// ==========================================
// Example 3: Closure with Parameters
// ==========================================
func multiplier(number int) func(int) int {

	return func(value int) int {
		return number * value
	}
}

// ==========================================
// Example 4: Generate Unique IDs
// ==========================================
func generateID() func() int {

	id := 1000

	return func() int {
		id++
		return id
	}
}

// ==========================================
// Main Function
// ==========================================
func main() {

	// ------------------------------------------------
	// Example 1 : Basic Closure
	// ------------------------------------------------
	fmt.Println("========== Example 1 : Basic Closure ==========")

	myFunction := outer()
	myFunction()

	fmt.Println()

	// ------------------------------------------------
	// Example 2 : Counter Using Closure
	// ------------------------------------------------
	fmt.Println("========== Example 2 : Counter ==========")

	increment := counter()

	fmt.Println(increment())
	fmt.Println(increment())
	fmt.Println(increment())
	fmt.Println(increment())

	fmt.Println()

	// ------------------------------------------------
	// Example 3 : Two Different Closures
	// ------------------------------------------------
	fmt.Println("========== Example 3 : Two Independent Closures ==========")

	c1 := counter()
	c2 := counter()

	fmt.Println("Counter 1")
	fmt.Println(c1())
	fmt.Println(c1())
	fmt.Println(c1())

	fmt.Println()

	fmt.Println("Counter 2")
	fmt.Println(c2())
	fmt.Println(c2())

	fmt.Println()

	// ------------------------------------------------
	// Example 4 : Closure with Parameters
	// ------------------------------------------------
	fmt.Println("========== Example 4 : Multiplier ==========")

	double := multiplier(2)
	triple := multiplier(3)

	fmt.Println("Double of 5 =", double(5))
	fmt.Println("Triple of 5 =", triple(5))

	fmt.Println()

	// ------------------------------------------------
	// Example 5 : Immediate Closure
	// ------------------------------------------------
	fmt.Println("========== Example 5 : Immediate Closure ==========")

	func() {
		fmt.Println("Hello from Immediate Closure")
	}()

	fmt.Println()

	// ------------------------------------------------
	// Example 6 : Generate Unique IDs
	// ------------------------------------------------
	fmt.Println("========== Example 6 : Generate ID ==========")

	nextID := generateID()

	fmt.Println(nextID())
	fmt.Println(nextID())
	fmt.Println(nextID())

	fmt.Println()

	// ------------------------------------------------
	// Example 7 : Closure Modifying Outer Variable
	// ------------------------------------------------
	fmt.Println("========== Example 7 : Modify Outer Variable ==========")

	value := 10

	show := func() {
		value += 5
		fmt.Println("Current Value:", value)
	}

	show()
	show()
	show()

	fmt.Println()

	// ------------------------------------------------
	// Example 8 : Closure Captures Variables
	// ------------------------------------------------
	fmt.Println("========== Example 8 : Capturing Variables ==========")

	name := "Alok"

	printName := func() {
		fmt.Println("Hello", name)
	}

	printName()

	name = "Rahul"

	printName()
}
