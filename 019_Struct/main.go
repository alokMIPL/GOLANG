package main

import (
	"fmt"
	"time"
)

// Structs are basically custom Data Structure

// Order Struct

type order struct {
	id        string
	amount    float32
	status    string
	createdAt time.Time
}

func main() {

	myOrder := order{
		id:     "1",
		amount: 50.00,
		status: "received",
	}

	// Add extra field in Struct
	myOrder.createdAt = time.Now()

	fmt.Println("Order Struct", myOrder)
	// Output = Order Struct {1 50 received {0 0 <nil>}}
	// Output = Order Struct {1 50 received {14023466436656812984 605001 0x7ff7080e4440}}

	// Now to get individual struct field
	fmt.Println("Order Struct", myOrder.id)
	// Output = Order Struct 1

}
