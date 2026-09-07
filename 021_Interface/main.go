package main

import "fmt"

type payment struct {
	gateway stripe
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

	stripePaymentGw := stripe{}
	newPayment := payment{
		gateway: stripePaymentGw,
	}

	newPayment.makePayment(100.00)
}
