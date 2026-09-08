Yes, that's right — you're just separating two different things: defining a type vs creating a value of that type.

1. The type definition (usually at package level, outside any function):

go
type stack struct {
	elements []int
}

This just tells Go "there exists a type called stack, and it has a field elements which is a slice of ints." It doesn't create any actual data in memory yet — it's a blueprint, like a class definition.

2. Creating a value of that type (this can go inside main, or any function):

go
myStack := stack{
	elements: []int{1, 2, 3},
}

Here you're actually instantiating that blueprint — creating a real stack value in memory, with elements set to [1, 2, 3].

You need step 1 to exist somewhere in the package (doesn't have to be right before main, Go doesn't care about declaration order at package level) before you can do step 2 anywhere that uses stack.

Think of it like this analogy:

go
type Person struct {
	Name string
}

func main() {
	p := Person{Name: "Alice"} // creating a value
	fmt.Println(p)
}

type Person struct {...} is the definition (can be top-level, outside main). p := Person{...} is you making an actual person value, and that line lives wherever you need it — often inside main or another function.

So your structure is already correct:

type stack struct { elements []int } → package-level, defines the shape
myStack := stack{elements: []int{1, 2, 3}} → inside main, creates an actual instance

One small note: in Go, struct types are usually capitalized (Stack) if you want them exported/visible outside the package, but for a main package that doesn't matter — lowercase stack is fine here.















# Go Generics Example

This project demonstrates the evolution from duplicated type-specific functions to using **Go generics**, along with type constraints (`any`, `comparable`, custom union constraints), and a generic `stack` type.

## What This Code Covers

### 1. The Problem: Code Duplication

We start with two nearly identical functions:

```go
func printSlice(items []int) {
	for _, item := range items {
		fmt.Println(item)
	}
}

func printStringSlice(items []string) {
	for _, item := range items {
		fmt.Println(item)
	}
}
```

Both do exactly the same thing — the only difference is the slice type (`[]int` vs `[]string`). This is the classic motivation for generics.

### 2. Generic Function with `any`

```go
func printGenericSlice[T any](items []T) {
	for _, item := range items {
		fmt.Println(item)
	}
}
```

- `T` is a **type parameter**.
- `any` means `T` can be **any type** (it's an alias for `interface{}`, but more readable).
- This single function replaces both `printSlice` and `printStringSlice`.

### 3. Generic Function with `comparable`

```go
func printGenericComparableSlice[T comparable](items []T) {
	for _, item := range items {
		fmt.Println(item)
	}
}
```

- `comparable` is a built-in constraint satisfied by all types that support `==` and `!=` — booleans, numbers, strings, pointers, channels, and structs/arrays composed entirely of comparable types.
- `comparable` can **only** be used as a type parameter constraint, not as a variable's type.

### 4. Scoped/Custom Type Constraint

```go
func printGenericScopedSlice[T int | string](items []T) {
	for _, item := range items {
		fmt.Println(item)
	}
}
```

- `int | string` restricts `T` to **only** `int` or `string`.
- Calling this with any other type (e.g. `float32`, `bool`) results in a **compile-time error**.

```go
// This would NOT compile:
// decimalNumber := []float32{1.9, 2.5, 3.3}
// printGenericScopedSlice(decimalNumber)
```

### 5. Non-Generic Stack

```go
type stack struct {
	elements []int
}
```

A basic stack type that can only ever hold `int` elements.

### 6. Generic Stack

```go
type genericStack[T any] struct {
	elements []T
}
```

By adding a type parameter `[T any]` to the struct itself, `genericStack` can hold **any** element type, decided at the time of instantiation:

```go
myGenericStack := genericStack[string]{
	elements: []string{"GOLANG", "JS", "SQL"},
}
```

## Running the Code

```bash
go run main.go
```

**Expected behavior:**
- Prints an `int` slice and a `string` slice using dedicated functions.
- Prints the same slices using `printGenericSlice` (works for any type).
- Prints the same slices using `printGenericScopedSlice` (restricted to `int`/`string`).
- Prints the same slices using `printGenericComparableSlice` (restricted to comparable types).
- Creates and prints a basic `stack` (int-only).
- Creates and prints a `genericStack[string]` (works with any type).

## Key Takeaways

| Concept | Meaning |
|---|---|
| `[T any]` | `T` can be any type at all |
| `[T comparable]` | `T` must support `==` / `!=` |
| `[T int \| string]` | `T` must be exactly `int` or `string` (union constraint) |
| Generic struct `genericStack[T any]` | The struct itself is parameterized, so one type definition supports many concrete types |

---

## Notes: Type Definition vs. Value Creation

A type definition and creating a value of that type are two separate steps in Go.

**1. The type definition** (usually at package level, outside any function):

```go
type stack struct {
	elements []int
}
```

This tells Go "there exists a type called `stack`, and it has a field `elements` which is a slice of ints." It doesn't create any actual data in memory yet — it's a blueprint, like a class definition.

**2. Creating a value of that type** (this can go inside `main`, or any function):

```go
myStack := stack{
	elements: []int{1, 2, 3},
}
```

Here you're actually instantiating that blueprint — creating a real `stack` value in memory, with `elements` set to `[1, 2, 3]`.

Step 1 needs to exist somewhere in the package (it doesn't have to be right before `main` — Go doesn't care about declaration order at the package level) before you can do step 2 anywhere that uses `stack`.

Think of it like this analogy:

```go
type Person struct {
	Name string
}

func main() {
	p := Person{Name: "Alice"} // creating a value
	fmt.Println(p)
}
```

`type Person struct {...}` is the definition (can be top-level, outside `main`). `p := Person{...}` is you making an actual person value, and that line lives wherever you need it — often inside `main` or another function.

So the structure in this project is correct:

- `type stack struct { elements []int }` → package-level, defines the shape
- `myStack := stack{elements: []int{1, 2, 3}}` → inside `main`, creates an actual instance

**One small note:** in Go, struct types are usually capitalized (`Stack`) if you want them exported/visible outside the package, but for a `main` package that doesn't matter — lowercase `stack` is fine here.
