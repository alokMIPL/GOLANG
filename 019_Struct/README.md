# Golang Structs — Order Example

This project demonstrates the fundamentals of **structs** in Go, including:

---

## What is a Struct?

A `struct` in Go is basically a custom data structure — a way to group related fields together under one type. It's similar to a "class" (without inheritance) in other languages.

```go
type order struct {
	id        string
	amount    float32
	status    string
	createdAt time.Time
}
```

Here, `order` groups four fields: `id`, `amount`, `status`, and `createdAt`.

---

## Methods on Structs (Receivers)

In Go, you attach functions to a struct using a **receiver**. This is how you "connect" a function to a struct type.

### Pointer Receiver — `(o *order)`

```go
func (o *order) changeStatus(status string) {
	o.status = status
}
```

- Used when you want to **modify** the original struct's data.
- Go automatically dereferences `o` for you, so you can write `o.status` instead of `(*o).status`.
- Since it's a pointer, changes made inside the method affect the **original** struct, not a copy.

### Value Receiver — `(o order)`

```go
func (o order) getAmount() float32 {
	return o.amount
}
```

- Used when you only want to **read** data, not modify it.
- Go passes a **copy** of the struct into the method, so any changes made inside won't affect the original.

### Rule of Thumb

| Receiver Type | Use When | Effect |
|---|---|---|
| `*order` (pointer) | You need to modify the struct | Changes original data |
| `order` (value) | You only need to read the struct | Works on a copy |

---

## Constructor Pattern in Go

Go doesn't have built-in constructors like other OOP languages. Instead, the convention is to write a regular function (usually prefixed with `new`) that returns a pointer to the struct:

```go
func newOrder(id string, amount float32, status string) *order {
	myOrder3 := order{
		id:     id,
		amount: amount,
		status: status,
	}
	return &myOrder3
}
```

Calling it:

```go
myOrderStruct := newOrder("1", 20.34, "Paid")
fmt.Println("myOrder3 of Struct by using Constructor", myOrderStruct)
// Output: &{1 20.34 Paid {0 0 <nil>}}
```

This returns a **pointer** to a newly created `order`, which is idiomatic in Go for constructor-like functions.

---

## Zero Values

If a struct field isn't explicitly set, Go automatically assigns it a **zero value** based on its type:

| Type | Zero Value |
|---|---|
| `string` | `""` |
| `int` / `float32` | `0` |
| `bool` | `false` |
| `time.Time` | `{0 0 <nil>}` (its own zero value) |
| pointers/slices/maps | `nil` |

This is why `createdAt` shows `{0 0 <nil>}` when it isn't explicitly initialized.

---

## Anonymous Structs

If you only need a struct **once** and don't want to formally define a named type, you can use an anonymous struct:

```go
language := struct {
	name string
	age  int
}{
	name: "Golang",
	age:  10,
}

fmt.Println(language)
// Output: {Golang 10}
```

This is useful for quick, one-off data groupings — for example, temporary API response shapes or test data.

---

## Full Code

```go
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

// To pass or connect the struct to a function
// receiver type
func (o *order) changeStatus(status string) {
	o.status = status
	// Here Go automatically dereferences the value.
	// Here we modify the value, so we use * (pointer)
}

func (o order) getAmount() float32 {
	return o.amount
	// Here we only read the value, so we don't need * (pointer)
}

// How to use a constructor in Go
// We achieve this by writing a regular function
func newOrder(id string, amount float32, status string) *order {
	myOrder3 := order{
		id:     id,
		amount: amount,
		status: status,
	}
	return &myOrder3
}

func main() {

	myOrder := order{
		id:     "1",
		amount: 50.00,
		status: "received",
	}

	// Add extra field to the struct after creation
	myOrder.createdAt = time.Now()

	fmt.Println("Order Struct", myOrder)
	// Output: Order Struct {1 50 received {14023466436656812984 605001 0x7ff7080e4440}}

	// Get an individual struct field
	fmt.Println("Order Struct", myOrder.id)
	// Output: Order Struct 1

	myOrder2 := order{
		id:        "2",
		amount:    40.00,
		status:    "delivered",
		createdAt: time.Now(),
	}

	fmt.Println("Order2 Struct", myOrder2)
	// Output: {2 40 delivered {14023467503996331940 669301 0x7ff631305440}}

	// Modify myOrder instance directly
	myOrder.status = "paid"
	fmt.Println("Order Struct", myOrder)
	// Output: Order Struct {1 50 paid {14023467791572242772 1 0x7ff714175440}}

	myOrder.changeStatus("Confirmed")
	// Passing the struct into a method to change its status
	fmt.Println("Order Struct by function", myOrder)
	// Output: Order Struct by function {1 50 Confirmed {14023469124807640180 1 0x7ff76ef55440}}

	fmt.Println(myOrder.getAmount())
	// Output: 50

	// If a struct field isn't set, Go assigns it a zero value by default.

	// Using the constructor function
	myOrderStruct := newOrder("1", 20.34, "Paid")
	fmt.Println("myOrder3 of Struct by using Constructor", myOrderStruct)
	// Output: myOrder3 of Struct by using Constructor &{1 20.34 Paid {0 0 <nil>}}

	// Anonymous struct — useful when you only need the struct once
	language := struct {
		name string
		age  int
	}{
		name: "Golang",
		age:  10,
	}

	fmt.Println(language)
	// Output: {Golang 10}

}
```