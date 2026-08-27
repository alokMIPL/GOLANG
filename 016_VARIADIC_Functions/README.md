# Variadic Functions in Golang

This project demonstrates **Variadic Functions** in Go (Golang). A variadic function allows you to pass **zero or more arguments** of the same type to a function. Inside the function, the variadic parameter behaves like a **slice**.

---

# What is a Variadic Function?

A **variadic function** is a function that can accept **any number of arguments** of the same type.

### Syntax

```go
func functionName(parameterName ...dataType) {
    // code
}
```

Example:

```go
func sum(numbers ...int) {
    // numbers is a slice ([]int)
}
```

---

# Project Structure

```
variadic-function/
│── main.go
│── README.md
```

---

# Examples Covered

This project includes the following examples:

- Basic Variadic Function (`sum`)
- Variadic Function with Strings (`printNames`)
- Finding the Maximum Number (`max`)
- Normal Parameter + Variadic Parameter (`greet`)
- Passing a Slice to a Variadic Function
- `fmt.Println()` as a Variadic Function
- Checking the Type of a Variadic Parameter

---

# Example 1: Basic Variadic Function

```go
func sum(numbers ...int)
```

This function:

- Accepts zero or more integers.
- Calculates their sum.
- Prints the slice and total.

Example:

```go
sum(10, 20)
sum(1, 2, 3, 4, 5)
sum()
```


Output:

```
Numbers: [10 20]
Sum: 30

Numbers: [1 2 3 4 5]
Sum: 15

Numbers: []
Sum: 0
```

---

# Example 2: Variadic Function with Strings

```go
func printNames(names ...string)
```

This function accepts multiple names and prints them.

Example:

```go
printNames("Alok")
printNames("Rahul", "Amit", "Priya")
```

Output:

```
Names:
1. Alok

Names:
1. Rahul
2. Amit
3. Priya
```

---

# Example 3: Find Maximum Number

```go
func max(nums ...int) int
```

This function returns the largest number from the provided integers.

Example:

```go
max(10, 50, 20, 90, 15)
```

Output:

```
Maximum Number: 90
```

---

# Example 4: Normal Parameter + Variadic Parameter

```go
func greet(message string, names ...string)
```

A variadic parameter **must always be the last parameter**.

Example:

```go
greet("Hello", "Alok", "Rahul", "Amit")
```

Output:

```
Hello Alok
Hello Rahul
Hello Amit
```

---

# Example 5: Passing a Slice

Suppose you already have a slice:

```go
numbers := []int{10, 20, 30, 40, 50}
```

❌ Wrong

```go
sum(numbers)
```

This produces a compile-time error.

✅ Correct

```go
sum(numbers...)
```

Output:

```
Numbers: [10 20 30 40 50]
Sum: 150
```

---

# Example 6: fmt.Println() is Variadic

The Go standard library uses variadic functions.

```go
fmt.Println("Hello")
fmt.Println("Hello", "World")
fmt.Println(10, 20, 30, 40, 50)
```

Its declaration is similar to:

```go
func Println(a ...any)
```

---

# Example 7: Variadic Parameter is a Slice

```go
func checkType(nums ...int)
```

Inside the function, the variadic parameter is actually a slice.

```go
fmt.Printf("%T", nums)
```

Output:

```
[]int
```

Example output:

```
Type of nums : []int
Length : 5
Values : [1 2 3 4 5]
```

---

# Rules of Variadic Functions

- A variadic function can accept zero or more arguments.
- All arguments must be of the same type.
- Inside the function, the variadic parameter behaves like a slice.
- A variadic parameter must always be the last parameter.
- Use the `...` operator to pass a slice to a variadic function.
- Only one variadic parameter is allowed in a function.

---

# Output

```
========== Example 1 : Sum ==========
Numbers: [10 20]
Sum: 30

Numbers: [1 2 3 4 5]
Sum: 15

Numbers: []
Sum: 0

========== Example 2 : Names ==========
Names:
1. Alok

Names:
1. Rahul
2. Amit
3. Priya

========== Example 3 : Maximum ==========
Maximum Number: 90

========== Example 4 : Greeting ==========
Hello Alok
Hello Rahul
Hello Amit

========== Example 5 : Slice ==========
Numbers: [10 20 30 40 50]
Sum: 150

========== Example 6 : fmt.Println ==========
Hello
Hello World
10 20 30 40 50

========== Example 7 : Type ==========
Type of nums : []int
Length : 5
Values : [1 2 3 4 5]
```

---

# Key Takeaways

- `...` makes a function variadic.
- Variadic functions accept any number of arguments.
- The variadic parameter is treated as a slice inside the function.
- A variadic parameter must be the last parameter.
- Use `slice...` to pass a slice to a variadic function.
- Many Go standard library functions, such as `fmt.Println()`, are variadic.

---

# Interview Questions

### 1. What is a variadic function?

A function that accepts **zero or more arguments** of the same type.

---

### 2. What is the type of a variadic parameter?

It is a **slice**.

Example:

```go
func demo(nums ...int)
```

Here, `nums` is of type `[]int`.

---

### 3. Can a function have multiple variadic parameters?

No.

---

### 4. Where should a variadic parameter be placed?

It must always be the **last parameter**.

---

### 5. How do you pass a slice to a variadic function?

```go
numbers := []int{1, 2, 3}

sum(numbers...)
```

Use the `...` operator to expand the slice into individual arguments.

---

# Author

**Alok Kumar**

Learning **Go (Golang)** from Beginner to Advanced 🚀