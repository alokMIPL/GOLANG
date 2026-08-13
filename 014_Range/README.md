# 🔁 Range in Go (Golang)

## What is `range`?

`range` is a keyword in Go used to iterate over different data structures.

It can be used with:

- Arrays
- Slices
- Strings
- Maps
- Channels

`range` automatically returns the index/key and value while looping.

---

# Syntax

```go
for index, value := range collection {
    // code
}
```

Example

```go
numbers := []int{10, 20, 30}

for index, value := range numbers {
    fmt.Println(index, value)
}
```

Output

```
0 10
1 20
2 30
```

---

# Range with Arrays

```go
package main

import "fmt"

func main() {

	arr := [5]int{10, 20, 30, 40, 50}

	for index, value := range arr {
		fmt.Println(index, value)
	}

}
```

Output

```
0 10
1 20
2 30
3 40
4 50
```

---

# Range with Slices

```go
package main

import "fmt"

func main() {

	numbers := []int{100, 200, 300, 400}

	for index, value := range numbers {
		fmt.Println(index, value)
	}

}
```

Output

```
0 100
1 200
2 300
3 400
```

---

# Ignoring Index

Use `_` when the index is not required.

```go
numbers := []int{10,20,30}

for _, value := range numbers {
	fmt.Println(value)
}
```

Output

```
10
20
30
```

---

# Ignoring Value

```go
numbers := []int{10,20,30}

for index := range numbers {
	fmt.Println(index)
}
```

Output

```
0
1
2
```

---

# Range with Strings

Strings are iterated character by character (Unicode runes).

```go
package main

import "fmt"

func main() {

	name := "Golang"

	for index, character := range name {
		fmt.Println(index, string(character))
	}

}
```

Output

```
0 G
1 o
2 l
3 a
4 n
5 g
```

---

# Unicode Example

```go
package main

import "fmt"

func main() {

	word := "नमस्ते"

	for index, character := range word {
		fmt.Println(index, string(character))
	}

}
```

Output

```
0 न
3 म
6 स
9 ् 
12 त
15 े
```

Notice the indexes increase according to UTF-8 byte positions.

---

# Range with Maps

```go
package main

import "fmt"

func main() {

	students := map[string]int{
		"Alice": 90,
		"Bob":   80,
		"John":  95,
	}

	for key, value := range students {
		fmt.Println(key, value)
	}

}
```

Possible Output

```
Alice 90
Bob 80
John 95
```

> **Note:** Map iteration order is **not guaranteed**.

---

# Ignoring Map Values

```go
students := map[string]int{
	"Alice":90,
	"Bob":80,
}

for key := range students {
	fmt.Println(key)
}
```

Output

```
Alice
Bob
```

---

# Ignoring Map Keys

```go
students := map[string]int{
	"Alice":90,
	"Bob":80,
}

for _, value := range students {
	fmt.Println(value)
}
```

Output

```
90
80
```

---

# Range with Channels

```go
package main

import "fmt"

func main() {

	ch := make(chan int)

	go func() {

		ch <- 10
		ch <- 20
		ch <- 30

		close(ch)

	}()

	for value := range ch {
		fmt.Println(value)
	}

}
```

Output

```
10
20
30
```

---

# Finding Sum Using Range

```go
numbers := []int{10,20,30,40}

sum := 0

for _, value := range numbers {
	sum += value
}

fmt.Println(sum)
```

Output

```
100
```

---

# Finding Maximum Value

```go
numbers := []int{12,45,7,90,34}

max := numbers[0]

for _, value := range numbers {

	if value > max {
		max = value
	}

}

fmt.Println(max)
```

Output

```
90
```

---

# Counting Even Numbers

```go
numbers := []int{1,2,3,4,5,6,7,8}

count := 0

for _, value := range numbers {

	if value%2==0{
		count++
	}

}

fmt.Println(count)
```

Output

```
4
```

---

# Modifying Slice Elements

```go
numbers := []int{1,2,3}

for index := range numbers{
	numbers[index] *=2
}

fmt.Println(numbers)
```

Output

```
[2 4 6]
```

---

# Nested Range Loop

```go
matrix := [][]int{
	{1,2},
	{3,4},
	{5,6},
}

for i,row := range matrix{

	for j,value := range row{

		fmt.Println(i,j,value)

	}

}
```

Output

```
0 0 1
0 1 2
1 0 3
1 1 4
2 0 5
2 1 6
```

---

# Common Operations

| Operation | Syntax |
|-----------|--------|
| Index + Value | `for i, v := range slice` |
| Only Value | `for _, v := range slice` |
| Only Index | `for i := range slice` |
| Map | `for k, v := range map` |
| String | `for i, ch := range string` |
| Channel | `for v := range channel` |

---

# When to Use `range`

Use `range` when:

- Reading slice elements
- Reading array elements
- Traversing strings
- Iterating over maps
- Reading values from channels
- Calculating sums
- Counting elements
- Searching values

---

# Difference Between Classic `for` and `range`

| Classic `for` | `range` |
|---------------|----------|
| Uses index manually | Index provided automatically |
| Best for custom increments | Best for iteration |
| More control | Cleaner syntax |
| Can loop infinitely | Only iterates collections |

Example

Classic for

```go
for i:=0;i<len(numbers);i++{
	fmt.Println(numbers[i])
}
```

Range

```go
for _,value:=range numbers{
	fmt.Println(value)
}
```

---

# Time Complexity

| Collection | Complexity |
|------------|------------|
| Array | O(n) |
| Slice | O(n) |
| Map | O(n) |
| String | O(n) |
| Channel | O(n) |

---

# Real-World Uses

- Reading database records
- Processing API responses
- Looping over JSON objects
- Counting words
- Reading CSV files
- Processing log files
- Reading configuration maps
- Traversing matrices
- Streaming channel data

---

# Complete Example

```go
package main

import "fmt"

func main() {

	numbers := []int{10,20,30,40,50}

	fmt.Println("Index and Value")

	for index,value:=range numbers{
		fmt.Println(index,value)
	}

	fmt.Println()

	fmt.Println("Only Value")

	for _,value:=range numbers{
		fmt.Println(value)
	}

	fmt.Println()

	fmt.Println("Only Index")

	for index:=range numbers{
		fmt.Println(index)
	}

}
```

Output

```
Index and Value
0 10
1 20
2 30
3 40
4 50

Only Value
10
20
30
40
50

Only Index
0
1
2
3
4
```

---

# Summary

- `range` is used to iterate over collections.
- Works with arrays, slices, strings, maps, and channels.
- Returns index/key and value.
- Use `_` to ignore unwanted values.
- Map iteration order is random.
- Strings iterate over Unicode runes.
- `range` makes loops cleaner and easier to read.

---

# 🎯 Practice Exercises

1. Print all elements of an array using `range`.
2. Find the sum of a slice.
3. Count even and odd numbers.
4. Find the maximum number in a slice.
5. Print each character of a string.
6. Iterate over a map of student marks.
7. Double every element in a slice using `range`.
8. Read values from a channel using `range`.
9. Traverse a 3×3 matrix using nested `range`.
10. Count the frequency of words in a slice using a map and `range`.

---

# 📖 Key Takeaways

- `range` is the preferred way to iterate over collections in Go.
- It simplifies loops by automatically providing indexes, keys, and values.
- It supports arrays, slices, maps, strings, and channels.
- Mastering `range` is essential before learning slices, maps, structs, concurrency, and Go's standard library.