package struct
// structs_demo.go
// A single-file, detailed tour of structs in Go.
// Run with: go run structs_demo.go

package main

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"
)

// -----------------------------------------------------------------------
// 1. Basic struct declaration
// -----------------------------------------------------------------------

type Person struct {
	Name string
	Age  int
	City string
}

// -----------------------------------------------------------------------
// 6. Nested struct
// -----------------------------------------------------------------------

type Address struct {
	Street string
	Zip    string
}

type Employee struct {
	Name    string
	Address Address
}

// -----------------------------------------------------------------------
// 7. Embedded struct (composition)
// -----------------------------------------------------------------------

type Animal struct {
	Name string
}

func (a Animal) Speak() string {
	return a.Name + " makes a sound"
}

type Dog struct {
	Animal // embedded, no field name -> fields/methods are "promoted"
	Breed  string
}

// -----------------------------------------------------------------------
// 8. Struct tags (used by encoding/json via reflection)
// -----------------------------------------------------------------------

type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email,omitempty"`
	pass  string `json:"-"` // unexported, never touched by json
}

// -----------------------------------------------------------------------
// 9. Methods with value vs pointer receivers
// -----------------------------------------------------------------------

type Rectangle struct {
	Width, Height float64
}

// Value receiver: operates on a copy, cannot mutate the original.
func (r Rectangle) Area() float64 {
	return r.Width * r.Height
}

// Pointer receiver: can mutate the original struct.
func (r *Rectangle) Scale(factor float64) {
	r.Width *= factor
	r.Height *= factor
}

// -----------------------------------------------------------------------
// 10. Comparable struct
// -----------------------------------------------------------------------

type Point struct {
	X, Y int
}

// A struct with a slice field is NOT comparable with ==.
type Bag struct {
	Items []string
}

// -----------------------------------------------------------------------
// 13. Constructor pattern (Go has no real constructors)
// -----------------------------------------------------------------------

func NewPerson(name string, age int) *Person {
	return &Person{Name: name, Age: age}
}

// -----------------------------------------------------------------------
// 15. Memory layout / alignment example
// -----------------------------------------------------------------------

type Bad struct {
	A bool
	B int64
	C bool
}

type Good struct {
	B int64
	A bool
	C bool
}

// -----------------------------------------------------------------------
// 16. Bonus: generic struct (Go 1.18+ type parameters)
// -----------------------------------------------------------------------

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	var zero T
	if len(s.items) == 0 {
		return zero, false
	}
	last := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return last, true
}

// -----------------------------------------------------------------------
// main: exercises every section above, in order
// -----------------------------------------------------------------------

func main() {
	fmt.Println("=== 1 & 2. Basic declaration & instantiation ===")
	p1 := Person{Name: "Alice", Age: 30, City: "Delhi"} // named fields
	p2 := Person{"Bob", 25, "Mumbai"}                    // positional
	var p3 Person                                        // zero value
	p4 := new(Person)                                     // *Person via new()
	fmt.Printf("p1=%+v\np2=%+v\np3(zero)=%+v\np4(new)=%+v\n\n", p1, p2, p3, p4)

	fmt.Println("=== 3. Accessing & modifying fields ===")
	fmt.Println("p1.Name before:", p1.Name)
	p1.Age = 31
	pp := &p1
	pp.Age = 32 // auto-dereferenced: same as (*pp).Age = 32
	fmt.Println("p1.Age after pointer mutation:", p1.Age)
	fmt.Println()

	fmt.Println("=== 4. Value semantics (copy on assignment) ===")
	p5 := p1
	p5.Name = "Charlie"
	fmt.Println("p1.Name:", p1.Name, "| p5.Name:", p5.Name, "(independent copies)")
	birthday(&p1)
	fmt.Println("p1.Age after birthday(&p1):", p1.Age)
	fmt.Println()

	fmt.Println("=== 5. Anonymous struct ===")
	point := struct {
		X, Y int
	}{X: 10, Y: 20}
	fmt.Printf("anonymous point=%+v\n\n", point)

	fmt.Println("=== 6. Nested struct ===")
	e := Employee{
		Name:    "Dev",
		Address: Address{Street: "MG Road", Zip: "201001"},
	}
	fmt.Println("e.Address.Street:", e.Address.Street)
	fmt.Println()

	fmt.Println("=== 7. Embedded struct (field/method promotion) ===")
	d := Dog{Animal: Animal{Name: "Rex"}, Breed: "Labrador"}
	fmt.Println("d.Name (promoted field):", d.Name)
	fmt.Println("d.Speak() (promoted method):", d.Speak())
	fmt.Println("d.Animal.Name (explicit):", d.Animal.Name)
	fmt.Println()

	fmt.Println("=== 8. Struct tags + JSON ===")
	u := User{ID: 1, Name: "Alice", pass: "secret"}
	b, _ := json.Marshal(u)
	fmt.Println("json.Marshal(u):", string(b))
	fmt.Println()

	fmt.Println("=== 9. Methods: value vs pointer receiver ===")
	r := Rectangle{Width: 3, Height: 4}
	fmt.Println("Area before scale:", r.Area())
	r.Scale(2)
	fmt.Println("Area after Scale(2):", r.Area())
	fmt.Println()

	fmt.Println("=== 10. Comparing structs ===")
	pt1 := Point{1, 2}
	pt2 := Point{1, 2}
	fmt.Println("pt1 == pt2:", pt1 == pt2)
	bag1 := Bag{Items: []string{"a"}}
	bag2 := Bag{Items: []string{"a"}}
	fmt.Println("bag1 == bag2 not possible with ==; reflect.DeepEqual:", reflect.DeepEqual(bag1, bag2))
	fmt.Println()

	fmt.Println("=== 11. Exported vs unexported fields ===")
	fmt.Println("(demonstrated only within this package; pass field is unexported)")
	fmt.Println()

	fmt.Println("=== 12. Empty struct as a set marker / signal ===")
	set := make(map[string]struct{})
	set["apple"] = struct{}{}
	_, exists := set["apple"]
	fmt.Println("apple in set:", exists)
	fmt.Println("size of struct{}:", unsafe.Sizeof(struct{}{}), "bytes")
	fmt.Println()

	fmt.Println("=== 13. Constructor pattern ===")
	newP := NewPerson("Alice", 30)
	fmt.Printf("NewPerson result: %+v (type %T)\n\n", newP, newP)

	fmt.Println("=== 15. Memory layout / alignment ===")
	fmt.Println("unsafe.Sizeof(Bad{}): ", unsafe.Sizeof(Bad{}), "bytes (padded)")
	fmt.Println("unsafe.Sizeof(Good{}):", unsafe.Sizeof(Good{}), "bytes (packed)")
	fmt.Println()

	fmt.Println("=== 16. Generic struct (Stack[T]) ===")
	var s Stack[int]
	s.Push(10)
	s.Push(20)
	s.Push(30)
	for {
		v, ok := s.Pop()
		if !ok {
			break
		}
		fmt.Println("popped:", v)
	}
}

func birthday(p *Person) {
	p.Age++
}
