# Golang Struct Embedding — Order & Customer Example

This project demonstrates **struct embedding** in Go — a way to compose structs together so one struct can reuse the fields of another, without classic inheritance.

---

## What is Struct Embedding?

Go doesn't support traditional class-based inheritance. Instead, it uses **composition** through **embedding**. You can place one struct inside another **without giving it a field name**, and its fields become directly accessible through the outer struct.

```go
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
```

Notice that `customer` is declared inside `order` **without a field name** — just the type name. This is what makes it an **embedded struct** (also called an "anonymous field").

---

## How Embedding Works

When a struct is embedded:

1. The outer struct (`order`) automatically gains access to all fields of the inner struct (`customer`).
2. You can access embedded fields in two ways:
   - **Promoted access:** `newOrder.name` (direct, as if it were `order`'s own field)
   - **Explicit access:** `newOrder.customer.name` (via the embedded struct's type name)
3. Internally, Go still stores it as a nested struct — embedding is really just **syntactic sugar** for field promotion.

```go
fmt.Println("Customer Name:", newOrder.customer.name)
// or equivalently
fmt.Println("Customer Name:", newOrder.name)
```

Both work because `name` is **promoted** from `customer` up to `order`.

---

## Full Code

```go
package main

import (
	"fmt"
	"time"
)

// We can embed a struct inside another struct to reuse its fields
// in the outer (parent) struct.

// Here, we embed the Customer struct inside the Order struct
// so we can use Customer's fields directly through Order.

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

	/*
		Output:
		Order Struct {1 50 received {0 0 <nil>} {John Doe 1234567890}}
		Order Struct {John Doe 1234567890}
		Customer Name: John Doe
		Customer Phone: 1234567890
	*/

}
```

---

## Output Breakdown

```
Order Struct {1 50 received {0 0 <nil>} {John Doe 1234567890}}
```

| Segment | Meaning |
|---|---|
| `1` | `id` |
| `50` | `amount` |
| `received` | `status` |
| `{0 0 <nil>}` | `createdAt` (zero value — not set in this example) |
| `{John Doe 1234567890}` | embedded `customer` struct (`name`, `phone`) |

```
Order Struct {John Doe 1234567890}
```
This line prints only the embedded `customer` field directly — `name` and `phone`.

```
Customer Name: John Doe
Customer Phone: 1234567890
```
Accessing fields explicitly through `newOrder.customer.name` and `newOrder.customer.phone`.

---

## Embedding vs. Named Field

| Approach | Syntax | Field Access |
|---|---|---|
| **Embedded struct** | `customer` (no field name) | `order.name` or `order.customer.name` (both work) |
| **Named field** | `cust customer` | Only `order.cust.name` (no promotion) |

Embedding is preferred when you want the outer struct to feel like it "has" all the inner struct's capabilities directly — similar to inheritance-style reuse, but built on **composition**.