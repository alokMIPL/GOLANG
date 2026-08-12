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
	whoAmI := func(i interface{}) {
		switch i.(type) {
		case int:
			fmt.Println("it's an integer")
		case string:
			fmt.Println("it's an string")
		case bool:
			fmt.Println("it's an boolean")
		case float32:
			fmt.Println("it's an float")
		default:
			fmt.Println("other")
		}
	}

	whoAmI(12)
	whoAmI("SWITCH Case")
	whoAmI(true)
	whoAmI(12.45)
	// whoAmI()

	/* Output of TYPE SWITCH
		it's workday
	it's an integer
	it's an string
	it's an boolean
	other

	*/

}
