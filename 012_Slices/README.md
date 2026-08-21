# 📚 Golang Slices (Introduction)

This program demonstrates the basic concepts of **Slices in Go (Golang)**.

A **slice** is a dynamically-sized, flexible view into the elements of an array. Unlike arrays, slices do not have a fixed size, making them one of the most commonly used data structures in Go.

> **Note:** Although slices are built on top of arrays, they provide much more flexibility because they can grow and shrink dynamically.

---

# Program Structure

```go
package main
```

Every executable Go program begins with the `main` package.

The `main` package tells the Go compiler that this file is an executable program.

---

```go
import "fmt"
```

The `fmt` package is imported to print output to the console.

In this program, only the following function is used:

```go
fmt.Println()
```

---

```go
func main() {
```

The `main()` function is the entry point of the Go program.

Execution starts from this function.

---

# 1. Slice Declaration

```go
var nums []int
```

## Explanation

This line declares an integer slice.

### Syntax

```go
var variableName []datatype
```

Here,

- Variable Name → `nums`
- Data Type → `int`

Notice that **no size is specified**.

Unlike arrays,

```go
var arr [4]int
```

a slice is declared as

```go
var nums []int
```

because slices are **dynamic**.

---

# Difference Between Array and Slice

## Array

```go
var arr [4]int
```

- Fixed size
- Length cannot change
- Memory is allocated immediately

---

## Slice

```go
var nums []int
```

- Dynamic size
- Can grow and shrink
- Initially has no underlying array

---

# 2. Printing the Slice

```go
fmt.Println(nums)
```

Output

```
[]
```

Although the output looks like an empty slice, it is **not an initialized empty slice**.

It is actually a **nil slice**.

---

# What is a Nil Slice?

A nil slice is a slice that has **not been initialized**.

It has:

- Length = 0
- Capacity = 0
- Underlying Array = nil

Visual representation

```
nums

Length   : 0
Capacity : 0
Pointer  : nil
```

Since no memory has been allocated yet, the slice points to **nil**.

---

# 3. Checking Whether the Slice is Nil

```go
fmt.Println(nums == nil)
```

Output

```
true
```

This confirms that `nums` is a **nil slice**.

---

# Why is the Output `true`?

When a slice is declared but not initialized,

```go
var nums []int
```

Go automatically sets it to its **zero value**.

The zero value of a slice is

```
nil
```

Therefore,

```go
nums == nil
```

returns

```
true
```

---

# Nil Slice vs Empty Slice

These two look similar but are different.

## Nil Slice

```go
var nums []int
```

Output

```
[]
```

Check

```go
nums == nil
```

Output

```
true
```

Properties

- Length = 0
- Capacity = 0
- Pointer = nil

---

## Empty Slice

```go
nums := []int{}
```

Output

```
[]
```

Check

```go
nums == nil
```

Output

```
false
```

Properties

- Length = 0
- Capacity = 0
- Memory has been allocated

Although both print

```
[]
```

they are **not the same**.

---

# Zero Value of a Slice

Every Go data type has a zero value.

| Data Type | Zero Value |
| --------- | ---------- |
| int       | 0          |
| bool      | false      |
| string    | ""         |
| pointer   | nil        |
| map       | nil        |
| slice     | nil        |
| channel   | nil        |
| function  | nil        |

The zero value of a slice is **nil**.

---

# Memory Representation

## Nil Slice

```
nums

Pointer
   │
   ▼
  nil

Length   = 0
Capacity = 0
```

No memory has been allocated.

---

## Empty Slice

```
nums

Pointer
   │
   ▼
+------+
|      |
+------+

Length   = 0
Capacity = 0
```

Memory exists even though there are no elements.

---

# Why are Slices Used?

Slices are the most commonly used collection type in Go because they provide:

- Dynamic size
- Easy insertion of elements
- Easy deletion of elements
- Efficient memory usage
- Convenient iteration
- Built-in helper functions

Most real-world Go programs use **slices** instead of arrays.

---

# Advantages of Slices

- Dynamic size
- Easy to pass to functions
- Backed by arrays
- Efficient memory usage
- Can grow automatically
- Supports built-in functions like `append()` and `copy()`

---

# Important Characteristics

- Dynamic data structure
- Reference type
- Backed by an underlying array
- Can grow automatically
- Zero value is `nil`
- Most frequently used collection in Go

---

# Complete Program

```go
package main

import "fmt"

func main() {

    // Uninitialized slice
    var nums []int

    fmt.Println(nums)

    fmt.Println(nums == nil)
}
```

---

# Output

```
[]
true
```

---

# Key Learnings

After completing this program, you understand:

- What a slice is.
- How to declare a slice.
- Why slices do not require a fixed size.
- What an uninitialized (nil) slice is.
- How to print a slice.
- How to check whether a slice is nil.
- The difference between a nil slice and an empty slice.
- Why slices are preferred over arrays in real-world Go applications.

---

# Next Topic

➡️ **Creating Slices using `make()`** – Learn how to initialize slices with a specific length and capacity, and how Go allocates memory for them.
