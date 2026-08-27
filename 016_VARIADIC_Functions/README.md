# Variadic Functions in Go

A simple Go program demonstrating **variadic functions** — functions that can accept a variable number of arguments.

## What is a Variadic Function?

A **variadic function** is a function that can be called with any number of arguments of a specified type. In Go, you declare one using `...` before the parameter type.

```go
func functionName(params ...Type) ReturnType
```

Inside the function, `params` behaves like a slice (`[]Type`) of all the arguments passed in.

---

## Code Overview

This program demonstrates two variadic functions:

| Function   | Signature              | Purpose                                              |
|------------|-------------------------|-------------------------------------------------------|
| `sum`      | `func sum(nums ...int) int` | Accepts any number of `int` arguments and returns their sum |
| `anyType`  | `func anyType(nums ...any) int` | Accepts any number of arguments of **any type** and returns the count |

It also shows that Go's built-in `fmt.Println()` is itself a variadic function.

---

## Full Source Code

```go
package main

import "fmt"

// how to make a variadic function
// Now this sum() function only takes int type
func sum(nums ...int) int {
	total := 0

	for _, num := range nums {
		total = total + num
	}
	return total
}

// If we want to make a function that take any Data Type then we need to use ...any or ...interface.

func anyType(nums ...any) int {
	fmt.Println(nums...)
	return len(nums)
}

func main() {

	// Variadic Function in fmt.Println()
	fmt.Println(1, 2, 3, 4, 5, 6, 7, "Hello")
	// In Variadic function we can pass any number of parameters, basically there is no limit for that.

	result := sum(1, 2, 3, 4, 5, 6, 7, 8)
	fmt.Println(result)

	finalResult := anyType(1, 3, 5, 6, "kola", 345, true, false, "alok")
	fmt.Println(finalResult)

}
```

---

## How It Works

### 1. `sum(nums ...int) int`

- Accepts **zero or more `int` values**.
- Inside the function, `nums` is treated as `[]int`.
- Uses a `for range` loop to add up every number and return the `total`.

```go
func sum(nums ...int) int {
	total := 0
	for _, num := range nums {
		total = total + num
	}
	return total
}
```

Calling `sum(1, 2, 3, 4, 5, 6, 7, 8)` sums all 8 integers and returns `36`.

### 2. `anyType(nums ...any) int`

- Accepts **any number of arguments of any type** (`int`, `string`, `bool`, etc.) using `...any`.
- `any` is a built-in alias for `interface{}` (introduced in Go 1.18) — it represents a type with zero method constraints, so it can hold a value of any type.
- `fmt.Println(nums...)` uses the `...` **spread operator** to unpack the slice and pass each element as an individual argument to `Println` (rather than printing the slice as one value).
- Returns `len(nums)` — the total count of arguments passed in.

```go
func anyType(nums ...any) int {
	fmt.Println(nums...)
	return len(nums)
}
```

Calling `anyType(1, 3, 5, 6, "kola", 345, true, false, "alok")` prints all 9 values and returns `9`.

### 3. `fmt.Println(...)` is Variadic Too

```go
fmt.Println(1, 2, 3, 4, 5, 6, 7, "Hello")
```

This works because `Println`'s signature is `func Println(a ...any) (n int, err error)` — it's already variadic, which is why you can pass mixed types and any number of them.

---

## Running the Program

```
1 2 3 4 5 6 7 Hello
36
1 3 5 6 kola 345 true false alok
9
```

| Line | Explanation |
|------|-------------|
| `1 2 3 4 5 6 7 Hello` | Output of the direct `fmt.Println(...)` call |
| `36` | Result of `sum(1,2,3,4,5,6,7,8)` → 1+2+3+4+5+6+7+8 |
| `1 3 5 6 kola 345 true false alok` | Printed inside `anyType`, showing all mixed-type values |
| `9` | Count of arguments passed to `anyType` |

---