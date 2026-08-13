package main

import "fmt"

func main() {

	// basic style
	var name string = "golang"
	fmt.Println(name)

	// short method
	/*
		var empName string
		short method it automatic infer or detect that the variable is string.
	*/

	var empName = "John"
	fmt.Println(empName)

	// Boolean
	var isAdult = true
	fmt.Println(isAdult)

	// Integer
	var age int = 12
	fmt.Println(age)

	// Now ShortHand method
	rollNumber := "02"
	fmt.Println(rollNumber)

	studentName := "Didarian"
	fmt.Println(studentName)

	// Declaration first and initialize later
	var carName string
	carName = "Honda"

	fmt.Println(carName)

	/* This Declaration first and initialize later
	will we use in every DataType like Int, Float, String etc.
	*/

}
