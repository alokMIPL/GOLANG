# Go Interfaces — Payment Gateway Example

This project demonstrates how Go **interfaces** enable polymorphism and dependency injection, using a simple payment-processing example with multiple gateways (Razorpay, Stripe, PayPal, and a fake/test gateway).

## What This Code Teaches

This is a classic **"program to an interface, not an implementation"** example. Instead of hardcoding a specific payment gateway inside the `payment` struct, the struct holds a `paymenter` interface field. This means:

- The `payment` struct doesn't need to know *which* gateway it's using.
- New gateways can be added without modifying `payment` at all.
- Swapping gateways (e.g., for testing) is as simple as changing which struct you assign to the interface field.

This pattern is the foundation of **dependency injection** and is heavily used for writing testable, decoupled code.

---

## Core Concept: Go Interfaces

```go
type paymenter interface {
	pay(amount float32)
}
```

- An interface in Go is a **set of method signatures**.
- Any type that implements all the methods of an interface **automatically satisfies** that interface — there's no `implements` keyword like in Java or C#.
- This is called **implicit satisfaction** (also known as "structural typing").

In this code, `razorpay`, `stripe`, `fakepayment`, and `paypal` all define a `pay(amount float32)` method, so they all satisfy `paymenter` without any explicit declaration.

---

## Code Walkthrough

### 1. The Interface

```go
type paymenter interface {
	pay(amount float32)
}
```

Defines a contract: "anything that wants to be a `paymenter` must have a `pay` method that takes a `float32` and returns nothing."

### 2. The `payment` Struct

```go
type payment struct {
	gateway paymenter
}
```

- Instead of storing a concrete gateway (like `stripe` or `razorpay` directly), it stores the `paymenter` **interface type**.
- This means `payment` can hold *any* value that satisfies `paymenter` — that's the whole point of using an interface here.

### 3. The `makePayment` Method

```go
func (p payment) makePayment(amount float32) {
	razorpayPaymentGw := razorpay{}
	razorpayPaymentGw.pay(amount)

	stripePaymentGw := stripe{}
	stripePaymentGw.pay(amount)

	p.gateway.pay(amount)
}
```

This method is called on a `payment` value and triggers a payment. (See [Known Issue](#known-issue-in-makepayment) below — this currently does more than intended.)

### 4. Concrete Gateway Implementations

Each gateway type implements the `pay` method, satisfying the `paymenter` interface:

```go
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

type paypal struct{}
func (p paypal) pay(amount float32) {
	fmt.Println("Making payment using PayPal", amount)
}
```

Each of these is a distinct, independent type. None of them know about each other, and none of them know about `payment` — they just implement `pay`.

### 5. The `main` Function

```go
func main() {
	paypalPaymentGw := paypal{}
	newPayment := payment{
		gateway: paypalPaymentGw,
	}

	newPayment.makePayment(100.00)
}
```

- Creates a `paypal{}` value.
- Injects it into `payment` via the `gateway` field.
- Calls `makePayment`, which (ideally) should route the payment through PayPal only.

The commented-out blocks above `paypalPaymentGw` show how easily you can swap in `stripe{}`, `razorpay{}`, or `fakepayment{}` instead — just change which struct is assigned to `gateway`. This is the core benefit of the interface-based design.

---

## How It Runs

Given the **current code as written**, running `go run main.go` produces:

```
Making payment using razorpay 100
Making payment using stripe 100
Making payment using PayPal 100
```

This happens because `makePayment` unconditionally calls both `razorpay` and `stripe`, **in addition to** whatever gateway was actually injected (`paypal`, in this case).

---

## Known Issue in `makePayment`

```go
func (p payment) makePayment(amount float32) {
	razorpayPaymentGw := razorpay{}
	razorpayPaymentGw.pay(amount)

	stripePaymentGw := stripe{}
	stripePaymentGw.pay(amount)

	p.gateway.pay(amount)
}
```

This defeats the purpose of the `paymenter` interface. No matter which gateway you configure via `payment.gateway`, you'll **always** also trigger `razorpay` and `stripe` payments. That means:

- You can never test with *only* `fakepayment`.
- You can never process a payment through *only* PayPal.
- Every payment silently triggers three gateway calls instead of one.

This is very likely a leftover from earlier iterations of the code before the interface was introduced, and it should be removed.

---

## Suggested Fix

```go
func (p payment) makePayment(amount float32) {
	p.gateway.pay(amount)
}
```

This is the entire point of using `paymenter`: `payment` doesn't need to know or care whether the underlying gateway is Razorpay, Stripe, PayPal, or a fake gateway used for testing. It just delegates to whatever was injected.

With this fix, running `main()` with `paypal` configured will correctly print only:

```
Making payment using PayPal 100
```

---