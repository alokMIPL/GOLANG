# Go Enums — `iota` vs String-based Constants

This project demonstrates how **enumerated types (enums)** are typically implemented in Go, since Go has no built-in `enum` keyword like other languages (Java, C#, TypeScript). Instead, Go uses **typed constants**, most commonly combined with the `iota` identifier.

---

## Table of Contents

1. [Why Go Doesn't Have Native Enums](#why-go-doesnt-have-native-enums)
2. [Approach 1: Integer Enum with `iota`](#approach-1-integer-enum-with-iota)
3. [Approach 2: String-based Enum](#approach-2-string-based-enum)
4. [Code Walkthrough](#code-walkthrough)
5. [How It Runs](#how-it-runs)
6. [Comparing the Two Approaches](#comparing-the-two-approaches)
7. [Known Limitation](#known-limitation)
8. [Suggested Improvement (Stringer Pattern)](#suggested-improvement-stringer-pattern)
9. [Key Takeaways](#key-takeaways)
10. [Possible Extensions](#possible-extensions)

---

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

Meanwhile, `changeOrderStatusString(PreparedString)` prints `Prepared` directly, because the underlying value *is* the string `"Prepared"`.

---

## Comparing the Two Approaches

| | **`iota` (int-based)** | **String-based** |
|---|---|---|
| Storage size | Small (int) — more memory efficient | Larger (string) |
| Readability when printed | Poor by default (prints raw number) | Good (prints readable text) |
| Performance | Faster comparisons (`int` comparison) | Slightly slower (`string` comparison) |
| Reordering safety | Risky — inserting a value in the middle shifts all subsequent values | Safe — each value is explicit and independent |
| Common use case | Internal state machines, status codes, flags | Logging, APIs, serialization (JSON), debugging |
| Requires extra work for readable output | Yes (need a `String()` method) | No — already human-readable |

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

---

## Key Takeaways

| Concept | Explanation |
|---|---|
| **No native `enum` keyword** | Go uses named types + `const` blocks to simulate enums. |
| **`iota`** | Auto-incrementing identifier, resets to `0` at each new `const` block, increments by 1 per line. |
| **Type safety** | A named type like `OrderStatus` prevents accidentally passing an unrelated `int`. |
| **Int enums** | Compact and fast, but not human-readable unless you add a `String()` method. |
| **String enums** | Human-readable by default, but use more memory and don't get auto-increment benefits. |
| **Stringer interface** | Implementing `String() string` lets `fmt` package print custom, readable output automatically. |

---

## Possible Extensions

1. **Add a `String()` method** to `OrderStatus` as shown above for readable debug/log output.
2. **Add validation**: a function like `func (s OrderStatus) IsValid() bool` to check if a value falls within the defined range.
3. **Use `iota` with bit-shifting** for flag-style enums (e.g., `1 << iota`) when values need to be combined with bitwise OR.
4. **JSON marshaling**: implement `MarshalJSON`/`UnmarshalJSON` on `OrderStatus` so it serializes as `"Prepared"` instead of `2` in APIs.
5. **Generate with `stringer` tool**: run `go install golang.org/x/tools/cmd/stringer@latest` and add `//go:generate stringer -type=OrderStatus` to auto-generate the `String()` method instead of writing it by hand.
6. **Combine both patterns**: keep `OrderStatus` as the canonical `int` type internally (efficient storage/comparison) but always expose it via `String()` for external-facing output — this is the most common idiomatic Go pattern.

---

*This README was generated to document and explain the accompanying Go source file for future review.*
