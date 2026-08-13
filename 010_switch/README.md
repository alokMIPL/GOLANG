# 🔀 Golang Switch Statement

This program demonstrates the different types of **Switch Statements** available in **Go (Golang)**.

A `switch` statement is used to execute one block of code among multiple possible conditions. It provides a cleaner and more readable alternative to writing multiple `if...else if...else` statements.

Go supports several types of switch statements, including:

- Simple Switch
- Multiple Case Switch
- Expression Switch
- Type Switch

---

# Program Structure

```go
package main
```

Every executable Go program starts with the `main` package.

The `main` package tells the Go compiler that this file contains the entry point of the application.

---

```go
import (
    "fmt"
    "time"
)
```

This program imports two packages.

### fmt

The `fmt` package is used for formatted input and output.

Functions used:

```go
fmt.Println()
```

---

### time

The `time` package provides functionality for working with dates and times.

In this program, it is used to determine the current day of the week.

---

```go
func main() {
```

The `main()` function is the starting point of the Go program.

Execution always begins from this function.

---

# What is a Switch Statement?

A switch statement compares a value against multiple cases.

Instead of writing

```go
if condition1 {

} else if condition2 {

} else if condition3 {

}
```

you can simply write

```go
switch value {

case value1:
    ...

case value2:
    ...

default:
    ...
}
```

Switch statements improve readability when handling many conditions.

---

# 1. Simple Switch Statement

```go
i := 5
```

A variable named `i` is declared and initialized with the value `5`.

---

```go
switch i {
```

The switch statement evaluates the value of `i`.

Go compares `i` with every `case` one by one.

---

```go
case 1:
    fmt.Println("One")
```

If `i == 1`

Output

```
One
```

---

```go
case 2:
    fmt.Println("two")
```

Runs only when

```
i == 2
```

---

```go
case 3:
    fmt.Println("three")
```

Runs only when

```
i == 3
```

---

```go
case 4:
    fmt.Println("four")
```

Runs only when

```
i == 4
```

---

```go
case 5:
    fmt.Println("five")
```

Since

```go
i := 5
```

this case matches.

Output

```
five
```

After executing the matching case, the switch statement automatically ends.

Unlike C, C++, or Java, Go does **not require a `break` statement**.

---

```go
default:
    fmt.Println("other")
```

The `default` block executes only if none of the cases match.

Example

```go
i := 20
```

Output

```
other
```

---

# Automatic Break in Go

In Go, every case automatically breaks after execution.

Example

```go
switch i {

case 1:
    fmt.Println("One")

case 2:
    fmt.Println("Two")
}
```

If `i` equals `1`, Go prints only

```
One
```

Execution does not continue to the next case.

---

# 2. Printing Separator

```go
fmt.Println("*****************")
```

This line is only used to separate different sections of the console output.

It has no effect on program logic.

---

# 3. Multiple Case Switch

```go
switch time.Now().Weekday() {
```

### Explanation

`time.Now()` returns the current date and time.

`Weekday()` returns the current day of the week.

Possible values include:

- Sunday
- Monday
- Tuesday
- Wednesday
- Thursday
- Friday
- Saturday

---

```go
case time.Saturday, time.Sunday:
```

A single case can contain multiple values separated by commas.

This means

```
Saturday OR Sunday
```

If today is Saturday or Sunday,

Output

```
it's weekend
```

---

```go
default:
    fmt.Println("it's workday")
```

If today is

- Monday
- Tuesday
- Wednesday
- Thursday
- Friday

Output

```
it's workday
```

---

# Why Use Multiple Cases?

Instead of writing

```go
case time.Saturday:
case time.Sunday:
```

Go allows

```go
case time.Saturday, time.Sunday:
```

This makes the code shorter and easier to read.

---

# 4. Type Switch

```go
whoAmI := func(i interface{}) {
```

This creates an anonymous function assigned to the variable `whoAmI`.

The parameter type is

```go
interface{}
```

`interface{}` (or `any` in Go 1.18+) can hold a value of **any type**.

Examples:

```go
12
```

```go
"Hello"
```

```go
true
```

```go
12.45
```

All of these can be passed to the function.

---

```go
switch i.(type) {
```

This is called a **Type Switch**.

Instead of comparing values, it compares the **data type** stored inside the interface.

---

### Integer Case

```go
case int:
```

Runs when the value is an integer.

Example

```go
whoAmI(12)
```

Output

```
it's an integer
```

---

### String Case

```go
case string:
```

Runs when the value is a string.

Example

```go
whoAmI("SWITCH Case")
```

Output

```
it's an string
```

---

### Boolean Case

```go
case bool:
```

Runs when the value is a boolean.

Example

```go
whoAmI(true)
```

Output

```
it's an boolean
```

---

### Float Case

```go
case float32:
```

Runs only when the value is of type `float32`.

Example

```go
var x float32 = 12.45
whoAmI(x)
```

Output

```
it's an float
```

---

### Default Case

```go
default:
```

Runs when none of the specified types match.

---

# Why Does

```go
whoAmI(12.45)
```

Print

```
other
```

The value

```go
12.45
```

is **not** of type `float32`.

Go treats decimal literals like `12.45` as **float64** by default.

Since your switch only checks

```go
case float32:
```

there is no matching case.

Therefore,

Output

```
other
```

---

# How to Detect float64?

You can add another case.

```go
case float64:
    fmt.Println("it's a float64")
```

Then

```go
whoAmI(12.45)
```

will print

```
it's a float64
```

---

# Function Calls

```go
whoAmI(12)
```

Output

```
it's an integer
```

---

```go
whoAmI("SWITCH Case")
```

Output

```
it's an string
```

---

```go
whoAmI(true)
```

Output

```
it's an boolean
```

---

```go
whoAmI(12.45)
```

Output

```
other
```

because `12.45` is `float64`.

---

# Complete Output

If today is a weekday, the output will be:

```
five

*****************

it's workday

it's an integer

it's an string

it's an boolean

other
```

If today is Saturday or Sunday, the output will be:

```
five

*****************

it's weekend

it's an integer

it's an string

it's an boolean

other
```

---

# Important Characteristics of Switch

- Cleaner than multiple `if...else` statements.
- Automatically breaks after a matching case.
- `break` is not required.
- Supports multiple values in a single case.
- Supports expressions.
- Supports type checking using Type Switch.
- Optional `default` case for unmatched conditions.

---

# Time Complexity

For a typical switch statement, Go checks cases sequentially until a match is found.

| Operation | Complexity |
|------------|------------|
| Best Case | O(1) |
| Worst Case | O(n) |

Where **n** is the number of cases.

---

# Key Learnings

After completing this program, you understand:

- What a switch statement is.
- How to write a simple switch.
- How `default` works.
- How Go automatically breaks after a matching case.
- How to use multiple values in a single `case`.
- How to use the `time` package with switch statements.
- What a Type Switch is.
- How `interface{}` can store values of any type.
- Why `12.45` is treated as `float64`.
- How Type Switch helps determine the runtime type of a value.

---

# Next Topic

➡️ **Go Loops (`for`, `range`)** – Learn how to iterate over arrays, slices, maps, strings, and channels using Go's only looping construct.