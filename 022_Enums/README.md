# Go Enums — `iota` vs String-based Constants

## Why Go Doesn't Have Native Enums

Unlike languages with a dedicated `enum` keyword, Go models enums using:

- A **named type** (e.g., `type OrderStatus int`)
- A **group of typed constants** representing the valid values

This gives you type safety (a function expecting `OrderStatus` won't silently accept a random `int`), but it means Go enums are really just "constants with a custom type" rather than a distinct language feature.

---

## Approach 1: Integer Enum with `iota`

```go
type OrderStatus int

const (
	Received OrderStatus = iota
	Confirmed
	Prepared
	Delivered
)
```

### What's happening here:

- `OrderStatus` is defined as a new type based on `int`.
- `iota` is a special Go identifier that **auto-increments** within a `const` block, starting at `0`.
- `Received = iota` → `0`
- `Confirmed` → `1` (inherits the same expression as the line above, with `iota` now `1`)
- `Prepared` → `2`
- `Delivered` → `3`

You don't have to manually assign `0, 1, 2, 3` — `iota` does it for you, and if you insert a new value in the middle, everything after it automatically shifts.

### Usage:

```go
func changeOrderStatus(status OrderStatus) {
	fmt.Println("Changing order status to", status)
}
```

This function only accepts `OrderStatus` values, so calling `changeOrderStatus(5)` (a raw int) won't compile — you must pass a valid `OrderStatus`, like `Prepared`.

---

## Approach 2: String-based Enum

```go
type OrderStatusString string

const (
	ReceivedString  OrderStatusString = "Received"
	ConfirmedString OrderStatusString = "Confirmed"
	PreparedString  OrderStatusString = "Prepared"
	DeliveredString OrderStatusString = "Delivered"
)
```

### What's happening here:

- `OrderStatusString` is a named type based on `string` instead of `int`.
- Each constant is explicitly assigned a readable string value.
- No `iota` is used here since each value needs its own explicit string — `iota` is primarily useful when values are sequential/numeric.

### Usage:

```go
func changeOrderStatusString(status OrderStatusString) {
	fmt.Println("Changing order status to = ", status, " = by using string enumerated type")
}
```

---

## Code Walkthrough

### `main` Function

```go
func main() {
	changeOrderStatus(Prepared)
	changeOrderStatusString(PreparedString)
}
```

- `changeOrderStatus(Prepared)` passes the integer-based enum value `Prepared` (which is `2` under the hood).
- `changeOrderStatusString(PreparedString)` passes the string-based enum value `PreparedString` (which is `"Prepared"`).

---

## How It Runs

Running `go run main.go` produces:

```
Changing order status to 2
Changing order status to =  Prepared  = by using string enumerated type
```

### Important observation:

Notice that `changeOrderStatus(Prepared)` prints `2`, **not** `"Prepared"`. This is because:

- `OrderStatus` is just an `int` under the hood.
- `fmt.Println` doesn't know how to convert `2` back into the label `"Prepared"` unless you tell it how (see [Suggested Improvement](#suggested-improvement-stringer-pattern) below).

Meanwhile, `changeOrderStatusString(PreparedString)` prints `Prepared` directly, because the underlying value _is_ the string `"Prepared"`.

---

## Comparing the Two Approaches

|                                         | **`iota` (int-based)**                                               | **String-based**                               |
| --------------------------------------- | -------------------------------------------------------------------- | ---------------------------------------------- |
| Storage size                            | Small (int) — more memory efficient                                  | Larger (string)                                |
| Readability when printed                | Poor by default (prints raw number)                                  | Good (prints readable text)                    |
| Performance                             | Faster comparisons (`int` comparison)                                | Slightly slower (`string` comparison)          |
| Reordering safety                       | Risky — inserting a value in the middle shifts all subsequent values | Safe — each value is explicit and independent  |
| Common use case                         | Internal state machines, status codes, flags                         | Logging, APIs, serialization (JSON), debugging |
| Requires extra work for readable output | Yes (need a `String()` method)                                       | No — already human-readable                    |

---

## Known Limitation

The `iota`-based `OrderStatus` type has a debugging drawback: printing an `OrderStatus` value just prints the underlying number (`2`), which isn't very useful when reading logs or debugging. You lose the readability that the string-based approach naturally provides.

Additionally, **if someone inserts a new constant in the middle of the `const` block**, all values afterward silently shift:

```go
const (
	Received OrderStatus = iota // 0
	Cancelled                   // 1  <-- newly inserted
	Confirmed                   // 2  <-- was 1, now shifted!
	Prepared                    // 3  <-- was 2, now shifted!
	Delivered                   // 4  <-- was 3, now shifted!
)
```

If these values are ever persisted (e.g., saved to a database as integers), this kind of reordering can silently corrupt existing data.

---

## Suggested Improvement (Stringer Pattern)

To get the best of both worlds — the efficiency of `int` and the readability of strings — implement the `Stringer` interface (`String() string`) on `OrderStatus`:

```go
func (s OrderStatus) String() string {
	switch s {
	case Received:
		return "Received"
	case Confirmed:
		return "Confirmed"
	case Prepared:
		return "Prepared"
	case Delivered:
		return "Delivered"
	default:
		return "Unknown"
	}
}
```

With this in place, `fmt.Println("Changing order status to", status)` will automatically call `status.String()` and print:

```
Changing order status to Prepared
```

This is the idiomatic Go way to make `iota`-based enums print nicely, and tools like `stringer` (part of `golang.org/x/tools`) can auto-generate this method for you from the `const` block.
