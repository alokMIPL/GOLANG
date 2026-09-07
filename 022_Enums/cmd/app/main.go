package main

import "fmt"

// enumerated types

func changeOrderStatus(status string) {
	fmt.Println("Changing order status to", status)
}

func main() {
	changeOrderStatus("received")
}
