# Closures in Go

A simple Go program demonstrating **closures** — functions that capture and retain access to variables from their surrounding scope, even after that outer scope has finished executing.

## What is a Closure?

A **closure** is a function value that references variables from outside its own body. The function "closes over" those variables — it can read and modify them, and they remain alive in memory for as long as the closure itself exists, even after the function that originally declared them has returned.

In Go, this is possible because functions are **first-class values**: they can be created inside other functions, returned as results, assigned to variables, and passed around like any other value (`int`, `string`, etc.).

---

## Full Source Code

```go
package main

import "fmt"

func counter() func() int {
	var count int = 0

	return func() int {
		count += 1
		return count
	}
}

func main() {
	increment := counter()
	fmt.Println(increment())
}
```

---

## Line-by-Line Explanation

### `func counter() func() int {`

This declares a function named `counter` that takes no arguments and **returns another function**. That returned function itself takes no arguments and returns an `int`. This return type — `func() int` — is what makes `counter` a **closure factory**: a function whose job is to produce closures.

### `var count int = 0`

A local variable `count`, initialized to `0`. Normally, a local variable like this would be destroyed the moment `counter()` finishes running and its stack frame is popped. But because the inner function (below) references `count`, Go detects this and keeps `count` alive on the **heap** instead of the stack, for as long as something still holds a reference to it.

### `return func() int { ... }`

This is an **anonymous function** (a function literal with no name) being returned from `counter`. It has direct access to `count` from the enclosing scope — this is the closure being formed. The anonymous function is now bundled together with its own private reference to `count`.

### `count += 1`

Each time this inner function is called, it increments the *same* `count` variable it captured — not a fresh copy. This is what gives the closure **state** that persists across calls.

### `return count`

Returns the current value of `count` after incrementing it.

### `increment := counter()`

Calling `counter()` runs the outer function once: it creates a new `count` variable (starting at `0`) and returns the inner function. That returned function is stored in the variable `increment`. At this point, `increment` **is** the closure — it's a function value permanently paired with its own `count`.

### `fmt.Println(increment())`

Calling `increment()` runs the inner function, which increments its captured `count` from `0` to `1` and returns `1`. `fmt.Println` then prints that returned value.

---

## How It Works Internally

```
counter() is called
        │
        ▼
┌─────────────────────────────┐
│ count := 0   (heap-allocated)│
│                              │
│ returns → func() int {      │
│              count += 1     │◄── this function "closes over" count
│              return count   │
│            }                 │
└─────────────────────────────┘
        │
        ▼
increment now holds:
  - the function body (count += 1; return count)
  - a reference to that specific count variable

Each call to increment():
  1st call → count: 0 → 1 → returns 1
  2nd call → count: 1 → 2 → returns 2
  3rd call → count: 2 → 3 → returns 3
```

The key insight: `count` is **not** reset to `0` on every call to `increment()`. It's only initialized once, when `counter()` runs. After that, `increment` keeps a persistent, private reference to that one `count` variable.

---

## Running the Program

1. Save the code in a file named `main.go`.
2. Make sure Go is installed ([https://go.dev/dl/](https://go.dev/dl/)).
3. Run it from your terminal:

```bash
go run main.go
```

---

## Expected Output

```
1
```

Since `increment()` is only called once in `main()`, the output is just `1` — the first increment of `count` from `0`.

If you called it multiple times, you'd see the counter persist and climb:

```go
func main() {
	increment := counter()
	fmt.Println(increment()) // 1
	fmt.Println(increment()) // 2
	fmt.Println(increment()) // 3
}
```

```
1
2
3
```

---

## Extending the Example

### Independent Counters

Each call to `counter()` creates a **brand-new** `count` variable. Two closures never share state unless you explicitly design them to:

```go
func main() {
	counterA := counter()
	counterB := counter()

	fmt.Println(counterA()) // 1
	fmt.Println(counterA()) // 2
	fmt.Println(counterB()) // 1 — independent from counterA
}
```

### A Decrementing / Resettable Counter

You can expand the closure to do more than one thing by returning multiple functions that share the same captured state:

```go
func counterWithReset() (increment func() int, reset func()) {
	count := 0

	increment = func() int {
		count += 1
		return count
	}

	reset = func() {
		count = 0
	}

	return increment, reset
}

func main() {
	inc, reset := counterWithReset()
	fmt.Println(inc())  // 1
	fmt.Println(inc())  // 2
	reset()
	fmt.Println(inc())  // 1 — back to start
}
```

Both `increment` and `reset` close over the **same** `count` variable, so they stay in sync.

---

## Common Pitfall

A frequent mistake with closures happens inside loops — capturing a loop variable can behave unexpectedly depending on your Go version:

```go
funcs := make([]func() int, 3)

for i := 0; i < 3; i++ {
	funcs[i] = counter() // fine — each call creates its own new closure
}
```

This is safe because `counter()` is called fresh each iteration, creating a distinct `count` each time. The pitfall applies more to code like:

```go
// Go < 1.22 : all three closures share the same `i`, printing 3, 3, 3
for i := 0; i < 3; i++ {
	funcs[i] = func() int { return i }
}
```

In **Go 1.22+**, loop variables are scoped per-iteration by default, so this specific issue no longer occurs — but it's worth knowing if you're reading or maintaining older Go code.

---