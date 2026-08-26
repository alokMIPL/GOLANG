# Go `range` — Iterating Over Data Structures

Notes and examples on using Go's `range` keyword to iterate over arrays,
slices, maps, and strings.

## Full Code

```go
package main

import "fmt"

// Iterating over data structures
func main() {

	nums := []int{6, 7, 8}

	// printing ARRAY by using FOR loop
	for i := 0; i < len(nums); i++ {
		fmt.Println(nums[i])
	}
	// Output = 6 7 8

	fmt.Println("***********")

	// Sum of Array by using RANGE

	sum := 0

	for _, num := range nums {
		sum = sum + num
		fmt.Println(num)
		// Output = 6 7 8
	}
	fmt.Println(sum)
	// Output = 21

	fmt.Println("***********")

	// Now to print number with INDEX so _, is replaced by i and i is INDEX in RANGE.

	// Now to access SLICE by RANGE
	for i, num := range nums {
		fmt.Println(num, i)
	}

	/* Output =
	6 0
	7 1
	8 2
	*/

	fmt.Println("***********")

	// Now access MAP by using RANGE

	m := map[string]string{"fname": "John", "lname": "doe"}

	for k, v := range m {
		fmt.Println(k, v)
	}

	fmt.Println("***********")

	// Now access String by using RANGE
	// Basically give UNICODE

	for i, c := range "golang" {
		fmt.Println(i, c)
	}

	/* Output =
	0 103
	1 111
	2 108
	3 97
	4 110
	5 103
	*/

	// Now getting each and every charater of STRING then use string() function.

	for i, c := range "golang" {
		fmt.Println(i, string(c))
	}

	/*
		0 g
		1 o
		2 l
		3 a
		4 n
		5 g
	*/

}
```

## Breakdown

### 1. Classic `for` loop over a slice
```go
for i := 0; i < len(nums); i++ {
	fmt.Println(nums[i])
}
```
Manual indexing — the traditional way to walk through a slice/array.

### 2. Summing with `range` (index discarded)
```go
for _, num := range nums {
	sum = sum + num
}
```
`range` returns `(index, value)` pairs. Using `_` discards the index when you
only need the value.

### 3. `range` with index and value
```go
for i, num := range nums {
	fmt.Println(num, i)
}
```
Keeps both the index (`i`) and the value (`num`) from each iteration.

### 4. Ranging over a `map`
```go
m := map[string]string{"fname": "John", "lname": "doe"}

for k, v := range m {
	fmt.Println(k, v)
}
```
`range` on a map yields `(key, value)` pairs. **Note:** map iteration order
in Go is not guaranteed — don't rely on it being the same every run.

### 5. Ranging over a `string` (byte index + rune)
```go
for i, c := range "golang" {
	fmt.Println(i, c)
}
```
`range` on a string iterates over Unicode code points (runes), not bytes.
`i` is the **byte offset** where that rune starts, and `c` is the rune's
integer (Unicode) value — which is why printing `c` directly shows numbers
like `103`, `111`, etc.

### 6. Converting the rune back to a character
```go
for i, c := range "golang" {
	fmt.Println(i, string(c))
}
```
Wrapping `c` in `string()` converts the rune back into its printable
character form (`g`, `o`, `l`, `a`, `n`, `g`).

## Key Takeaways

| Data structure | `range` returns |
|---|---|
| Array / Slice | `index, value` |
| Map | `key, value` |
| String | `byte index, rune (int32)` |

- Use `_` to discard whichever part of the pair you don't need.
- For strings, remember `range` gives you **runes**, not bytes — use
  `string(c)` to get the actual character.
- Map iteration order is randomized by Go's runtime; don't depend on it.
