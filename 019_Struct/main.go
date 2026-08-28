// structs_tutorial.go
// A single-file, fully detailed tour of structs in Go.
// Run with: go run structs_tutorial.go
package main

import (
	"encoding/json"
	"fmt"
)

// ---------------------------------------------------------------------
// 1. BASIC STRUCT DEFINITION
// ---------------------------------------------------------------------
type Person struct {
	Name string
	Age  int
	City string
}

// ---------------------------------------------------------------------
// 2. STRUCT WITH TAGS (used by encoding/json, validators, ORMs, etc.)
// ---------------------------------------------------------------------
type Employee struct {
	ID       int     `json:"id"`
	Name     string  `json:"name"`
	Salary   float64 `json:"salary,omitempty"`
	password string  // unexported field -> not visible outside package, not marshaled
}

// ---------------------------------------------------------------------
// 3. NESTED STRUCT
// ---------------------------------------------------------------------
type Address struct {
	Street string
	City   string
	Zip    string
}

type Customer struct {
	Name    string
	Address Address // nested struct (composition)
}

// ---------------------------------------------------------------------
// 4. STRUCT EMBEDDING (Go's version of inheritance)
// ---------------------------------------------------------------------
type Animal struct {
	Name string
}

func (a Animal) Speak() string {
	return a.Name + " makes a sound"
}

type Dog struct {
	Animal // embedded struct -> Dog "inherits" fields & methods
	Breed  string
}

// Dog can override the embedded method
func (d Dog) Speak() string {
	return d.Name + " barks"
}

// ---------------------------------------------------------------------
// 5. METHODS: VALUE RECEIVER vs POINTER RECEIVER
// ---------------------------------------------------------------------
type Counter struct {
	count int
}

// Value receiver -> operates on a COPY, original is unchanged
func (c Counter) IncrementCopy() {
	c.count++
}

// Pointer receiver -> operates on the ORIGINAL, mutation persists
func (c *Counter) Increment() {
	c.count++
}

// ---------------------------------------------------------------------
// 6. ANONYMOUS STRUCTS (no named type, used for quick throwaway data)
// ---------------------------------------------------------------------
func anonymousStructExample() {
	point := struct {
		X, Y int
	}{X: 3, Y: 4}
	fmt.Println("Anonymous struct:", point)
}

// ---------------------------------------------------------------------
// 7. STRUCT COMPARISON (structs are comparable if all fields are comparable)
// ---------------------------------------------------------------------
func compareExample() {
	p1 := Person{"Alice", 30, "NYC"}
	p2 := Person{"Alice", 30, "NYC"}
	fmt.Println("p1 == p2:", p1 == p2) // true, field-by-field comparison
}

// ---------------------------------------------------------------------
// 8. CONSTRUCTOR-STYLE FUNCTION (Go has no real constructors)
// ---------------------------------------------------------------------
func NewEmployee(id int, name string, salary float64) *Employee {
	return &Employee{ID: id, Name: name, Salary: salary}
}

// ---------------------------------------------------------------------
// MAIN: demonstrates everything above
// ---------------------------------------------------------------------
func main() {
	// --- Basic struct ---
	p := Person{Name: "John", Age: 25, City: "Delhi"}
	fmt.Println("Basic struct:", p)
	fmt.Println("Access field:", p.Name, p.Age)

	// Struct literal without field names (order matters)
	p2 := Person{"Sara", 28, "Mumbai"}
	fmt.Println("Positional literal:", p2)

	// Zero-value struct
	var p3 Person
	fmt.Println("Zero value struct:", p3) // {"" 0 ""}

	// --- Pointers to structs ---
	pp := &p
	pp.Age = 26 // Go auto-dereferences: same as (*pp).Age = 26
	fmt.Println("Modified via pointer:", p)

	// --- JSON tags in action ---
	emp := NewEmployee(101, "Nina", 55000)
	jsonBytes, _ := json.Marshal(emp)
	fmt.Println("JSON output:", string(jsonBytes))

	// --- Nested struct ---
	cust := Customer{
		Name: "Ravi",
		Address: Address{
			Street: "MG Road",
			City:   "Bengaluru",
			Zip:    "560001",
		},
	}
	fmt.Println("Nested struct:", cust)
	fmt.Println("Nested field access:", cust.Address.City)

	// --- Embedding ---
	d := Dog{Animal: Animal{Name: "Rex"}, Breed: "Labrador"}
	fmt.Println("Embedded field access:", d.Name) // promoted field
	fmt.Println("Overridden method:", d.Speak())
	fmt.Println("Embedded method directly:", d.Animal.Speak())

	// --- Value vs pointer receiver ---
	c := Counter{}
	c.IncrementCopy()
	fmt.Println("After IncrementCopy (unchanged):", c.count) // 0
	c.Increment()
	fmt.Println("After Increment (changed):", c.count) // 1

	// --- Slice of structs ---
	people := []Person{
		{"A", 20, "X"},
		{"B", 22, "Y"},
	}
	for i, person := range people {
		fmt.Printf("people[%d] = %+v\n", i, person)
	}

	// --- Map of structs ---
	inventory := map[string]Employee{
		"e1": {ID: 1, Name: "Tom", Salary: 40000},
	}
	fmt.Println("Map of struct:", inventory["e1"])

	// --- Anonymous struct ---
	anonymousStructExample()

	// --- Struct comparison ---
	compareExample()
}
