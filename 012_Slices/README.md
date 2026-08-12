# What are Slices?

Slices are **dynamic arrays** in Go that provide more flexibility than traditional arrays. They are one of the most important data structures you'll use in Go programming.

### Key Characteristics:

- ✅ **Dynamic size** - Can grow and shrink
- ✅ **Reference type** - Point to an underlying array
- ✅ **Built-in functions** - `append()`, `copy()`, `len()`, `cap()`
- ✅ **Zero value** - `nil` (no underlying array)
- ✅ **Flexible** - More powerful than arrays

---

## Arrays vs Slices

| Feature | Arrays | Slices |
|---------|--------|--------|
| **Size** | Fixed (compile-time) | Dynamic (runtime) |
| **Type** | `[n]T` | `[]T` |
| **Zero value** | Zeroed elements | `nil` |
| **Pass to function** | Copy | Reference |
| **Memory** | Stack (often) | Heap |
| **Flexibility** | ❌ Limited | ✅ High |

---

## Code Examples

### Creating Slices

```go
// Method 1: Using make()
slice := make([]int, 5, 10)  // length: 5, capacity: 10

// Method 2: Slice literal
slice := []int{1, 2, 3, 4, 5}

// Method 3: From array
arr := [5]int{1, 2, 3, 4, 5}
slice := arr[1:4]  // Creates slice from index 1 to 3

// Method 4: nil slice
var slice []int  // nil slice, no underlying array