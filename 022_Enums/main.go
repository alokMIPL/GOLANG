package main

import "fmt"

// enumerated types

type OrderStatus int

const (
	Received OrderStatus = iota
	Confirmed
	Prepared
	Delivered
)

func changeOrderStatus(status OrderStatus) {
	fmt.Println("Changing order status to", status)
}

// If we want to use string in enumerated types, we can use the following approach
type OrderStatusString string

const (
	ReceivedString  OrderStatusString = "Received"
	ConfirmedString OrderStatusString = "Confirmed"
	PreparedString  OrderStatusString = "Prepared"
	DeliveredString OrderStatusString = "Delivered"
)

func changeOrderStatusString(status OrderStatusString) {
	fmt.Println("Changing order status to = ", status, " = by using string enumerated type")
}

func main() {
	changeOrderStatus(Prepared)
	changeOrderStatusString(PreparedString)
}
