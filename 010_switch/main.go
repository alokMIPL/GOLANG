package main

import (
	"fmt"
	"time"
)

func main() {

	// When we have too much conditions then we use SWITCH.

	// So Simple Switch

	i := 5

	switch i {
	case 1:
		fmt.Println("One")
	case 2:
		fmt.Println("two")
	case 3:
		fmt.Println("three")
	case 4:
		fmt.Println("four")
	case 5:
		fmt.Println("five")
	default:
		fmt.Println("other")
	}

	fmt.Println("*****************")

	// Multiple condition Switch

	switch time.Now().Weekday() {
	case time.Saturday, time.Sunday:
		fmt.Println("it's weekend")
	default:
		fmt.Println("it's workday")
	}

	// TYPE SWITCH

}
