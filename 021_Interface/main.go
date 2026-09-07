package main

import "fmt"

type payment struct {
	// first we use strip as payment gateway
	// gateway stripe

	// Now we want to use razorpay as payment gateway so we can use interface here
	gateway razorpay
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

func main() {
	// newPayment := payment{}

	// stripePaymentGw := stripe{}
	// newPayment := payment{
	// 	gateway: stripePaymentGw,
	// }

	razorpayPaymentGw := razorpay{}
	newPayment := payment{
		gateway: razorpayPaymentGw,
	}

	newPayment.makePayment(100.00)
}
