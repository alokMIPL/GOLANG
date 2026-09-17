package main

import "fmt"

func add(a int, b int) int {
	return a + b
}

func SumAndProduct(a int, b int) (int, int) {
	sum := a + b
	product := a * b
	return sum, product
}

func main() {

	sum1 := add(10, 20)

	fmt.Println("SUM1 answer is = ", sum1)

	// Output = SUM1 answer is =  30

	s, p := SumAndProduct(12, 34)

	fmt.Println("Sum and Product result = ", s, p)
	// Output = Sum and Product result =  46 408

}
