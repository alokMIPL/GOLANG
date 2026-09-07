package main

import (
	"fmt"
	"time"
)

// We can embed struct in another struct to use the fields of the embedded struct in the parent struct

// Like in Order struct we can embed the Customer struct to use the fields of Customer in Order struct

type customer struct {
	name  string
	phone string
}

type order struct {
	id        string
	amount    float32
	status    string
	createdAt time.Time
	customer  // Embedded struct
}

func main() {

	newOrder := order{
		id:     "1",
		amount: 50.00,
		status: "received",
		customer: customer{
			name:  "John Doe",
			phone: "1234567890",
		},
	}

	fmt.Println("Order Struct", newOrder)
	fmt.Println("Order Struct", newOrder.customer)
	fmt.Println("Customer Name:", newOrder.customer.name)
	fmt.Println("Customer Phone:", newOrder.customer.phone)
	// Output = Order Struct {1 50 received {0 0 <nil>} {John Doe 1234567890}}

	/*
		Order Struct {1 50 received {0 0 <nil>} {John Doe 1234567890}}
		Order Struct {John Doe 1234567890}
		Customer Name: John Doe
		Customer Phone: 1234567890
	*/

}
