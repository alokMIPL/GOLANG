# Closures in Golang

This project demonstrates **Closures** in Go (Golang). A closure is a function that **captures and remembers variables from its outer scope**, even after the outer function has finished executing.

Closures are commonly used for maintaining state, creating counters, generating IDs, callbacks, and writing cleaner, reusable code.

---

# What is a Closure?

A **closure** is an anonymous function that has access to variables declared outside of it.

Unlike a normal function, a closure **remembers** the variables from the surrounding scope even after the outer function has returned.

### Syntax

```go
func outerFunction() func() {

	value := 10

	return func() {
		fmt.Println(value)
	}
}
```

---

# Project Structure

```
closures/
│── main.go
│── README.md
```

---

# Examples Covered

This project demonstrates the following closure examples:

- Basic Closure
- Counter Using Closure
- Two Independent Closures
- Closure with Parameters
- Immediate (Anonymous) Closure
- Generate Unique IDs
- Modifying Outer Variables
- Capturing Variables from Outer Scope

---

# Example 1: Basic Closure

```go
func outer() func()
```

The outer function creates a variable and returns an anonymous function.

The anonymous function remembers the variable even after the outer function finishes.

Example:

```go
myFunction := outer()
myFunction()
```

Output

```
Hello Golang
```

---

# Example 2: Counter Using Closure

```go
func counter() func() int
```

This closure maintains the value of a counter between function calls.

Example:

```go
increment := counter()

increment()
increment()
increment()
increment()
```

Output

```
1
2
3
4
```

The variable `count` is preserved inside the closure.

---

# Example 3: Two Independent Closures

Each call to `counter()` creates a completely new closure.

Example

```go
c1 := counter()
c2 := counter()
```

Output

```
Counter 1
1
2
3

Counter 2
1
2
```

Both closures have their own independent `count` variable.

---

# Example 4: Closure with Parameters

```go
func multiplier(number int) func(int) int
```

The outer function receives a number and returns a function that multiplies another number.

Example

```go
double := multiplier(2)
triple := multiplier(3)

double(5)
triple(5)
```

Output

```
Double of 5 = 10
Triple of 5 = 15
```

---

# Example 5: Immediate Closure

A closure can be created and executed immediately.

Example

```go
func() {
	fmt.Println("Hello from Immediate Closure")
}()
```

Output

```
Hello from Immediate Closure
```

This type of closure is commonly used for one-time execution.

---

# Example 6: Generate Unique IDs

Closures are useful for maintaining state.

Example

```go
nextID := generateID()

nextID()
nextID()
nextID()
```

Output

```
1001
1002
1003
```

The variable `id` is remembered after every function call.

---

# Example 7: Modifying an Outer Variable

Closures can change variables declared outside the function.

Example

```go
value := 10

show := func() {
	value += 5
	fmt.Println(value)
}
```

Output

```
15
20
25
```

The updated value is stored and reused during the next call.

---

# Example 8: Capturing Variables

Closures capture variables, not copies of values.

Example

```go
name := "Alok"

printName := func() {
	fmt.Println(name)
}

printName()

name = "Rahul"

printName()
```

Output

```
Hello Alok
Hello Rahul
```

The closure uses the latest value of the variable.

---

# How Closures Work

```
Outer Function
      │
      ▼
Creates Variable

count = 0

      │
      ▼
Returns Anonymous Function

      │
      ▼
Anonymous Function Remembers Variable

Call 1 → 1
Call 2 → 2
Call 3 → 3
Call 4 → 4
```

---

# Output

```
========== Example 1 : Basic Closure ==========
Hello Golang

========== Example 2 : Counter ==========
1
2
3
4

========== Example 3 : Two Independent Closures ==========
Counter 1
1
2
3

Counter 2
1
2

========== Example 4 : Multiplier ==========
Double of 5 = 10
Triple of 5 = 15

========== Example 5 : Immediate Closure ==========
Hello from Immediate Closure

========== Example 6 : Generate ID ==========
1001
1002
1003

========== Example 7 : Modify Outer Variable ==========
Current Value: 15
Current Value: 20
Current Value: 25

========== Example 8 : Capturing Variables ==========
Hello Alok
Hello Rahul
```

---

# Advantages of Closures

- Preserve state between function calls.
- Reduce the need for global variables.
- Improve code readability.
- Create reusable functions.
- Generate counters and unique IDs.
- Useful for callbacks and middleware.
- Help encapsulate private data.

---

# Rules of Closures

- A closure is usually an anonymous function.
- A closure can access variables from the outer function.
- It remembers those variables even after the outer function has returned.
- Each closure created separately has its own state.
- Closures can modify outer variables.
- Closures capture variables, not copies of values.

---

# Real-World Uses

Closures are commonly used for:

- Counters
- Unique ID generators
- Caching
- Authentication middleware
- Callback functions
- Event handlers
- HTTP handlers
- Custom function generators

---

# Interview Questions

### 1. What is a closure?

A closure is an anonymous function that captures and remembers variables from its surrounding scope.

---

### 2. Why do we use closures?

Closures help maintain state without using global variables and make code more modular and reusable.

---

### 3. Does a closure remember variables after the outer function returns?

Yes. The captured variables remain available as long as the closure exists.

---

### 4. Can a closure modify an outer variable?

Yes.

Example

```go
count++
```

The updated value is remembered for the next call.

---

### 5. Are two closures created from the same function independent?

Yes.

```go
c1 := counter()
c2 := counter()
```

Each closure has its own independent state.

---

### 6. What is an immediate closure?

A closure that is defined and executed immediately.

Example

```go
func() {
	fmt.Println("Hello")
}()
```

---

### 7. Do closures capture values or variables?

Closures capture **variables**, not copies of their values.

---

# Key Takeaways

- A closure is a function that remembers variables from its outer scope.
- Closures preserve state between function calls.
- Every closure has its own independent state.
- Closures can modify outer variables.
- Closures are widely used for counters, generators, callbacks, and middleware.
- Closures make Go programs more flexible and maintainable.

---

# Author

**Alok Kumar**

Learning **Go (Golang)** from **Beginner → Advanced** 🚀