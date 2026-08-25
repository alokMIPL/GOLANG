# Go Maps — Notes & Examples

This document explains the concepts demonstrated in `main.go`, which covers the basics of working with **maps** in Go.

A **map** is Go's built-in key-value data structure — equivalent to a `hash`, `object`, or `dict` in other programming languages.

---

## 1. Creating a Map

Maps can be created using the built-in `make()` function:

```go
m := make(map[string]string)
```

This creates an empty map with `string` keys and `string` values.

---

## 2. Setting Elements

You can add or update key-value pairs using simple assignment:

```go
m["name"] = "golang"
m["area"] = "backend"
```

---

## 3. Getting Elements

Access a value using its key:

```go
fmt.Println(m["name"], m["area"])
// Output: golang backend
```

---

## 4. Accessing a Non-Existent Key

If you try to access a key that doesn't exist in the map, Go does **not** throw an error — it returns the **zero value** for the map's value type.

```go
fmt.Println(m["phone"])
// Output: (empty string, since zero value of string is "")
```

### Zero Value Depends on Type

| Value Type | Zero Value |
|------------|------------|
| `string`   | `""`       |
| `int`      | `0`        |
| `bool`     | `false`    |

Example with `int`:

```go
n := make(map[string]int)
n["roll"] = 12
fmt.Println(n["roll"], n["kola"])
// Output: 12 0
```

Here, `"kola"` was never set, so accessing it returns `0` (the zero value for `int`), not an error.

---

## 5. Getting the Length of a Map

Use the built-in `len()` function to get the number of key-value pairs:

```go
fmt.Println(len(m), len(n))
// Output: 2 1
```

---

## 6. Printing the Whole Map

You can print the entire map directly:

```go
fmt.Println(m)
// Output: map[area:backend name:golang]
```

> Note: Go prints map keys in **sorted order** when using `fmt.Println`, not insertion order.

---

## 7. Deleting an Element

Use the built-in `delete()` function to remove a key-value pair:

```go
delete(m, "area")
fmt.Println(m)
// Output: map[name:golang]
```

---

## 8. Clearing / Emptying a Map

Use the built-in `clear()` function to remove all elements from a map at once:

```go
clear(m)
fmt.Println(m)
// Output: map[] (empty map)
```

---

## 9. Declaring a Map with a Map Literal

Instead of using `make()`, you can declare and initialize a map in one step using a **map literal**:

```go
shop := map[string]int{"price": 40, "phones": 3}
fmt.Println(shop)
// Output: map[phones:3 price:40]
```

---

## 10. Checking if a Key Exists (Comma-OK Idiom)

Go provides a special two-value form for map access, commonly called the **"comma-ok" idiom**, to check whether a key exists:

```go
v, ok := shop["phones"]
fmt.Println(v)
if ok {
	fmt.Println("all ok")
} else {
	fmt.Println("not ok")
}
// Output:
// 3
// all ok
```

- `v` → the value for the key (zero value if not found)
- `ok` → a `bool` indicating whether the key actually exists

This is the safe way to distinguish between "key exists with zero value" and "key doesn't exist at all."

---

## 11. Comparing Two Maps

Maps **cannot** be compared directly using `==` in Go (this would cause a compile error). Instead, use `maps.Equal()` from the standard library `maps` package (introduced in Go 1.21+).

```go
import "maps"

m1 := map[string]int{"price": 40, "phones": 3}
m2 := map[string]int{"price": 40, "phones": 3}

fmt.Println(maps.Equal(m1, m2))
// Output: true
```

If the values differ, it returns `false`:

```go
m3 := map[string]int{"price": 140, "phones": 3}
m4 := map[string]int{"price": 40, "phones": 3}

fmt.Println(maps.Equal(m3, m4))
// Output: false
```

`maps.Equal` returns `true` only if both maps have the same length and all key-value pairs match exactly.

---

## Summary of Built-in Functions Used

| Function       | Purpose                                    |
|----------------|---------------------------------------------|
| `make()`       | Create an empty map                        |
| `len()`        | Get number of key-value pairs              |
| `delete()`     | Remove a specific key-value pair           |
| `clear()`      | Remove all key-value pairs                 |
| `maps.Equal()` | Compare two maps for equality (needs `maps` package) |

## Key Takeaways

- Accessing a missing key never panics — it returns the zero value of the value type.
- Use the **comma-ok idiom** (`v, ok := m[key]`) when you need to know whether a key truly exists.
- Map literals (`map[K]V{...}`) are a quicker alternative to `make()` + assignment when initial values are known.
- Maps print with keys in sorted order via `fmt.Println`.
- Direct `==` comparison isn't allowed for maps — use `maps.Equal()` instead.
