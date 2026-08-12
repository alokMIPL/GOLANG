# 📦 Golang Arrays

This program demonstrates the basic concepts of **Arrays in Go (Golang)**.

An array is a collection of elements of the **same data type** stored in contiguous memory locations. Unlike slices, the size of an array is fixed and cannot be changed after it is created.

---

# Program Structure

```go
package main
```

- Every executable Go program starts with `package main`.
- The `main` package tells the Go compiler that this file contains the entry point of the application.

---

```go
import "fmt"
```

The `fmt` package is imported because it provides functions for formatted input and output.

Examples:

- `fmt.Println()`
- `fmt.Printf()`
- `fmt.Sprintf()`

In this program, only `fmt.Println()` is used.

---

```go
func main() {
```

The `main()` function is the starting point of every Go application.

Execution always begins from this function.

---

# 1. Array Declaration

```go
var nums [4]int
```

## Explanation

This line declares an integer array.

### Syntax

```go
var variableName [size]datatype
```

Here,

- Variable Name → `nums`
- Size → `4`
- Data Type → `int`

So,

```
Index : 0 1 2 3
Value : 0 0 0 0
```

The array contains **4 integer elements**.

Every element is automatically initialized to **0** because integers have a zero value of `0`.

---

# 2. Finding Array Length

```go
fmt.Println(len(nums))
```

The built-in function `len()` returns the total number of elements present in the array.

Output

```
4
```

Even though no values have been assigned yet, the length remains **4** because the size is fixed during declaration.

---

# 3. Printing Separator

```go
fmt.Println("*****************")
```

This line is only used to make the console output easier to read.

It has no effect on the program logic.

---

# 4. Assigning a Value

```go
nums[0] = 1
```

This assigns the value `1` to the first element of the array.

Arrays use **zero-based indexing**.

| Index | Value |
|-------|------|
|0|1|
|1|0|
|2|0|
|3|0|

---

# 5. Accessing an Element

```go
fmt.Println(nums[0])
```

This prints the value stored at index `0`.

Output

```
1
```

---

# 6. Printing the Entire Array

```go
fmt.Println(nums)
```

Output

```
[1 0 0 0]
```

Only the first element has been assigned.

The remaining elements still contain their default value (`0`).

---

# Why Doesn't Go Store Garbage Values?

Unlike languages like C or C++, Go automatically initializes every variable.

This is called the **Zero Value**.

Therefore,

```
[1 0 0 0]
```

instead of

```
[1 ? ? ?]
```

---

# 7. Boolean Array

```go
var vals [4]bool
```

Creates a boolean array of size 4.

Since no values are assigned,

Output

```
[false false false false]
```

The default value of `bool` is

```
false
```

---

# 8. String Array

```go
var names [4]string
```

Creates an array of four strings.

Output

```
[   ]
```

Each element is actually an empty string.

Internally it looks like

```
["" "" "" ""]
```

The empty string is the zero value for the `string` type.

---

# 9. Assigning a String

```go
names[2] = "GOLANG"
```

Only index 2 receives a value.

Array becomes

```
["" "" "GOLANG" ""]
```

Printing

```go
fmt.Println(names)
```

Output

```
[  GOLANG ]
```

Notice that only the third position contains a value.

---

# 10. Array Initialization During Declaration

Instead of assigning values one by one,

```go
arr := [3]int{1,2,3}
```

creates and initializes the array in one statement.

Output

```
[1 2 3]
```

---

## Rules of Initialization

### Correct

```go
arr := [3]int{1,2,3}
```

---

### Less Elements

```go
arr := [5]int{1,2}
```

Output

```
[1 2 0 0 0]
```

Remaining positions receive zero values.

---

### More Elements

```go
arr := [3]int{1,2,3,4}
```

Compiler Error

```
index 3 is out of bounds
```

because an array of size 3 can store only three elements.

---

# 11. Two-Dimensional Array

```go
arrays := [2][2]int{{3,4},{5,6}}
```

This creates a matrix.

```
[
 [3 4]
 [5 6]
]
```

Rows = 2

Columns = 2

Output

```
[[3 4] [5 6]]
```

You can access elements using

```go
arrays[0][1]
```

Output

```
4
```

---

# 12. When Should Arrays Be Used?

Arrays are useful when:

### Fixed Size

The number of elements never changes.

Example

- Days in a Week
- Months in a Year
- Chess Board

---

### Memory Optimization

Since the size is fixed, Go allocates memory only once.

No resizing is required.

---

### Constant Time Access

Every element has an index.

Accessing

```go
nums[3]
```

takes constant time

```
O(1)
```

regardless of array size.

---

# Zero Values in Go

| Data Type | Zero Value |
|------------|------------|
| int | 0 |
| float64 | 0 |
| bool | false |
| string | "" |
| pointer | nil |
| interface | nil |
| slice | nil |
| map | nil |
| channel | nil |
| function | nil |

---

# Important Characteristics of Arrays

- Fixed Size
- Same Data Type
- Contiguous Memory Allocation
- Fast Index Access
- Size is Part of the Type
- Zero Value Initialization
- Cannot Grow Dynamically

---

# Time Complexity

| Operation | Complexity |
|------------|------------|
| Access by Index | O(1) |
| Update | O(1) |
| Traverse | O(n) |
| Search | O(n) |

---

# Memory Representation

```
Index

0      1      2      3
+------+------+------+------+
|  1   |  0   |  0   |  0   |
+------+------+------+------+
```

Memory is allocated continuously.

---

# Complete Output

```
4

*****************

1

*****************

[1 0 0 0]

*****************

[false false false false]

*****************

[   ]

*****************

[  GOLANG ]

*****************

[1 2 3]

*****************

[[3 4] [5 6]]
```

---

# Key Learnings

After completing this program, you understand:

- How to declare arrays.
- How to initialize arrays.
- How to access elements using indexes.
- How Go assigns default (zero) values.
- How to print arrays.
- How to find array length.
- How to work with string and boolean arrays.
- How to initialize arrays during declaration.
- How to create two-dimensional arrays.
- When arrays should be preferred over slices.