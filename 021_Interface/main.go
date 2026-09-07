package main

import "fmt"

// Now Interface is a type that defines a set of method signatures. A value of interface type can hold any value that implements those methods. In Go, interfaces are satisfied implicitly, meaning that a type implements an interface simply by implementing its methods.

type paymenter interface {
	pay(amount float32)
}

type payment struct {
	// first we use strip as payment gateway
	// gateway stripe

	// Now we want to use razorpay as payment gateway so we can use interface here
	// gateway razorpay

	// Now we want to use fake payment gateway for testing so we can use interface here
	gateway paymenter
}

func (p payment) makePayment(amount float32) {
	razorpayPaymentGw := razorpay{}
	razorpayPaymentGw.pay(amount)

	stripePaymentGw := stripe{}
	stripePaymentGw.pay(amount)

	p.gateway.pay(amount)
}

type razorpay struct{}

func (r razorpay) pay(amount float32) {
	fmt.Println("Making payment using razorpay", amount)
}

type stripe struct{}

func (s stripe) pay(amount float32) {
	fmt.Println("Making payment using stripe", amount)
}

type fakepayment struct{}

func (f fakepayment) pay(amount float32) {
	fmt.Println("Making payment using fake payment gateway", amount)
}

// Now adding PayPal as payment gateway
type paypal struct{}

func (p paypal) pay(amount float32) {
	fmt.Println("Making payment using PayPal", amount)
}

func main() {
	// newPayment := payment{}

	// stripePaymentGw := stripe{}
	// newPayment := payment{
	// 	gateway: stripePaymentGw,
	// }

	// razorpayPaymentGw := razorpay{}
	// newPayment := payment{
	// 	gateway: razorpayPaymentGw,
	// }

	// fakeGw := fakepayment{}
	// newPayment := payment{
	// 	gateway: fakeGw,
	// }

	paypalPaymentGw := paypal{}
	newPayment := payment{
		gateway: paypalPaymentGw,
	}

	newPayment.makePayment(100.00)
}
