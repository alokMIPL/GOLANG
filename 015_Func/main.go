package main

import "fmt"

// ======================================================
// Simple Function (No Parameters, No Return)
// ======================================================

func greet() {
	fmt.Println("Welcome to Golang!")
}

// ======================================================
// Function with Parameters
// ======================================================

func sayHello(name string) {
	fmt.Println("Hello,", name)
}

// ======================================================
// Function Returning a Value
// ======================================================

func add(a, b int) int {
	return a + b
}

// ======================================================
// Function Returning Multiple Values
// ======================================================

func calculate(a, b int) (int, int) {
	sum := a + b
	product := a * b
	return sum, product
}

// ======================================================
// Named Return Values
// ======================================================

func rectangle(length, width int) (area int, perimeter int) {
	area = length * width
	perimeter = 2 * (length + width)
	return
}

// ======================================================
// Variadic Function
// ======================================================

func sum(numbers ...int) int {

	total := 0

	for _, value := range numbers {
		total += value
	}

	return total
}

// ======================================================
// Recursive Function
// ======================================================

func factorial(n int) int {

	if n == 0 || n == 1 {
		return 1
	}

	return n * factorial(n-1)
}

// ======================================================
// Function as Parameter
// ======================================================

func operate(a, b int, operation func(int, int) int) int {
	return operation(a, b)
}

func multiply(a, b int) int {
	return a * b
}

// ======================================================
// Returning a Function
// ======================================================

func getMultiplier() func(int) int {

	return func(x int) int {
		return x * 2
	}

}

// ======================================================
// Struct
// ======================================================

type Student struct {
	Name string
	Age  int
}

// ======================================================
// Method (Function with Receiver)
// ======================================================

func (s Student) display() {
	fmt.Println("Name:", s.Name)
	fmt.Println("Age :", s.Age)
}

// ======================================================
// Main Function
// ======================================================

func main() {

	// ===============================================
	// 1. Simple Function
	// ===============================================

	fmt.Println("========== SIMPLE FUNCTION ==========")

	greet()

	// ===============================================
	// 2. Function with Parameter
	// ===============================================

	fmt.Println("\n========== PARAMETERS ==========")

	sayHello("Alok")
	sayHello("Rahul")

	// ===============================================
	// 3. Return Value
	// ===============================================

	fmt.Println("\n========== RETURN VALUE ==========")

	result := add(15, 25)

	fmt.Println("Addition =", result)

	// ===============================================
	// 4. Multiple Return Values
	// ===============================================

	fmt.Println("\n========== MULTIPLE RETURN ==========")

	sumValue, product := calculate(10, 5)

	fmt.Println("Sum =", sumValue)
	fmt.Println("Product =", product)

	// ===============================================
	// 5. Named Return Values
	// ===============================================

	fmt.Println("\n========== NAMED RETURN ==========")

	area, perimeter := rectangle(10, 5)

	fmt.Println("Area =", area)
	fmt.Println("Perimeter =", perimeter)

	// ===============================================
	// 6. Variadic Function
	// ===============================================

	fmt.Println("\n========== VARIADIC FUNCTION ==========")

	fmt.Println(sum(10, 20, 30))
	fmt.Println(sum(5, 10, 15, 20, 25))

	// Slice to Variadic Function

	numbers := []int{1, 2, 3, 4, 5}

	fmt.Println(sum(numbers...))

	// ===============================================
	// 7. Anonymous Function
	// ===============================================

	fmt.Println("\n========== ANONYMOUS FUNCTION ==========")

	func() {
		fmt.Println("This is an Anonymous Function")
	}()

	// ===============================================
	// 8. Function Stored in Variable
	// ===============================================

	fmt.Println("\n========== FUNCTION VARIABLE ==========")

	subtract := func(a, b int) int {
		return a - b
	}

	fmt.Println(subtract(30, 10))

	// ===============================================
	// 9. Recursive Function
	// ===============================================

	fmt.Println("\n========== RECURSION ==========")

	fmt.Println("Factorial of 5 =", factorial(5))

	// ===============================================
	// 10. Function as Parameter
	// ===============================================

	fmt.Println("\n========== FUNCTION AS PARAMETER ==========")

	answer := operate(6, 7, multiply)

	fmt.Println(answer)

	// ===============================================
	// 11. Returning Function
	// ===============================================

	fmt.Println("\n========== RETURNING FUNCTION ==========")

	double := getMultiplier()

	fmt.Println(double(10))
	fmt.Println(double(50))

	// ===============================================
	// 12. Method
	// ===============================================

	fmt.Println("\n========== METHOD ==========")

	student := Student{
		Name: "Alok",
		Age:  26,
	}

	student.display()

	// ===============================================
	// 13. Defer
	// ===============================================

	fmt.Println("\n========== DEFER ==========")

	defer fmt.Println("This prints last.")

	fmt.Println("First")
	fmt.Println("Second")

	// ===============================================
	// 14. Local Variable Scope
	// ===============================================

	fmt.Println("\n========== VARIABLE SCOPE ==========")

	showScope()

	// ===============================================
	// 15. Even or Odd
	// ===============================================

	fmt.Println("\n========== EVEN OR ODD ==========")

	checkEvenOdd(11)
	checkEvenOdd(20)

	// ===============================================
	// 16. Maximum of Two Numbers
	// ===============================================

	fmt.Println("\n========== MAXIMUM ==========")

	fmt.Println(max(10, 50))

	// ===============================================
	// 17. Swap Values
	// ===============================================

	fmt.Println("\n========== SWAP ==========")

	a, b := swap(100, 200)

	fmt.Println(a, b)

	// ===============================================
	// 18. Power Function
	// ===============================================

	fmt.Println("\n========== POWER ==========")

	fmt.Println(power(2, 5))

	fmt.Println("\n========== END ==========")
}

// ======================================================
// Local Scope Example
// ======================================================

func showScope() {

	x := 100

	fmt.Println("Local Variable =", x)

}

// ======================================================
// Even or Odd
// ======================================================

func checkEvenOdd(number int) {

	if number%2 == 0 {
		fmt.Println(number, "is Even")
	} else {
		fmt.Println(number, "is Odd")
	}

}

// ======================================================
// Maximum
// ======================================================

func max(a, b int) int {

	if a > b {
		return a
	}

	return b

}

// ======================================================
// Swap
// ======================================================

func swap(a, b int) (int, int) {
	return b, a
}

// ======================================================
// Power
// ======================================================

func power(base, exponent int) int {

	result := 1

	for i := 1; i <= exponent; i++ {
		result *= base
	}

	return result

}
