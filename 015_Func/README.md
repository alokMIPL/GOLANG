# Go Functions — First-Class Citizens, Multiple Returns & Higher-Order Functions

Notes and examples covering Go function basics: parameter/return typing,
returning multiple values, and treating functions as first-class values.

## Full Code

```go
package main

import "fmt"

/*
In this line "func add(a int, b int) int {"

this line represent the input taken by function add(a int, b int) and type is int.

but this int { define that there is a return type and must be int.

*/

// Function 1
func add(a int, b int) int {
	// we can also write as
	// add(a, b int)
	return a + b
}

// Function 2
// We can return multiple values in GOLANG Function.
func getLanguages() (string, string, string) {
	return "golang", "javaScript", "c++"
}

// Function 3
// We can return multiple values in GOLANG Function.
func getLanguagesMixed() (string, string, bool) {
	return "golang", "javaScript", true
}

// Basically in GOLANG we retun two things the function value and the secind one is error.

// To compress any compiler error we use "_" to supress that error.

// GOLANG functions are first class citizen functions() and can be used or assigned those functions to any variable and pass those functions as arguments for other functions.

func apply(a, b int, op func(int, int) int) int {
	return op(a, b)
}

func adds(x, y int) int {
	return x + y
}

func multiply(x, y int) int {
	return x * y
}

func main() {
	result := add(3, 5)
	fmt.Println(result)
	// Output = 8
	fmt.Println(getLanguages())
	// Output = golang javaScript c++

	fmt.Println(getLanguagesMixed())
	// Output = golang javaScript true

	// Assign a function to a variable
	greet := func(name string) string {
		return "Hello, " + name
	}

	fmt.Println(greet("Alok"))
	// Output = Hello, Alok

	fmt.Println(apply(3, 4, add))
	fmt.Println(apply(3, 4, multiply))
	// Output = 7 12

}
```

## Breakdown

### 1. Basic function signature
```go
func add(a int, b int) int {
	return a + b
}
```
- `(a int, b int)` — the parameters and their types.
- The trailing `int` before `{` is the **return type**; the function must
  return an `int`.
- Shorthand: when consecutive parameters share the same type, you can write
  `func add(a, b int) int` instead of repeating `int` for each one.

### 2. Returning multiple values
```go
func getLanguages() (string, string, string) {
	return "golang", "javaScript", "c++"
}
```
Go functions can return more than one value — no need for a struct, tuple,
or wrapper object. The return types are listed in parentheses, matching the
order of the returned values.

```go
func getLanguagesMixed() (string, string, bool) {
	return "golang", "javaScript", true
}
```
Mixed types work the same way — each position in the return list has its
own declared type.

> **Common Go convention:** many standard library functions return
> `(value, error)` — e.g. `result, err := someFunc()`. If you don't care
> about one of the returned values, discard it with `_`:
> ```go
> _, err := someFunc()
> ```

### 3. Functions as first-class values
```go
greet := func(name string) string {
	return "Hello, " + name
}
fmt.Println(greet("Alok")) // Hello, Alok
```
A function can be assigned to a variable just like any other value, then
called through that variable.

### 4. Passing functions as arguments (higher-order functions)
```go
func apply(a, b int, op func(int, int) int) int {
	return op(a, b)
}
```
`apply` accepts another function (`op`) as a parameter, with the signature
`func(int, int) int`. Any function matching that signature can be passed in:

```go
func adds(x, y int) int {
	return x + y
}

func multiply(x, y int) int {
	return x * y
}
```
```go
fmt.Println(apply(3, 4, add))      // 7
fmt.Println(apply(3, 4, multiply)) // 12
```

Note: in the `main` output, `apply(3, 4, add)` uses the `add` function
defined earlier (not `adds`, which is unused in `main` in this file) —
both work the same way since they share the same `func(int, int) int`
signature.

## Key Takeaways

| Concept | Example |
|---|---|
| Typed parameters | `func add(a int, b int) int` |
| Shorthand shared type | `func add(a, b int) int` |
| Multiple return values | `func f() (string, string, bool)` |
| Discard a return value | `_, err := f()` |
| Assign function to variable | `greet := func(name string) string {...}` |
| Pass function as argument | `func apply(a, b int, op func(int, int) int) int` |

- Go doesn't require you to bundle multiple outputs into a struct — multiple
  return values are a first-class language feature.
- The `(value, error)` return pattern is idiomatic Go for error handling,
  and `_` is the standard way to ignore a value you don't need.
- Any function whose signature matches a parameter's function type
  (e.g. `func(int, int) int`) can be passed in — this is how Go achieves
  higher-order functions without special syntax.
