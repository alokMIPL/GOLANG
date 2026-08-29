package 
// structs_advanced.go
// Part 2: more practical struct patterns in Go.
// Run with: go run structs_advanced.go

package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// -----------------------------------------------------------------------
// 1. Struct implementing an interface (polymorphism without inheritance)
// -----------------------------------------------------------------------

type Shape interface {
	Area() float64
	Perimeter() float64
}

type Circle struct {
	Radius float64
}

func (c Circle) Area() float64      { return 3.14159 * c.Radius * c.Radius }
func (c Circle) Perimeter() float64 { return 2 * 3.14159 * c.Radius }

type Square struct {
	Side float64
}

func (s Square) Area() float64      { return s.Side * s.Side }
func (s Square) Perimeter() float64 { return 4 * s.Side }

func describe(s Shape) {
	fmt.Printf("  %T -> Area=%.2f Perimeter=%.2f\n", s, s.Area(), s.Perimeter())
}

// -----------------------------------------------------------------------
// 2. Custom String() method (fmt.Stringer) — controls how a struct prints
// -----------------------------------------------------------------------

type Money struct {
	Cents int
}

func (m Money) String() string {
	return fmt.Sprintf("$%d.%02d", m.Cents/100, m.Cents%100)
}

// -----------------------------------------------------------------------
// 3. Struct implementing the error interface — custom error types
// -----------------------------------------------------------------------

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on %q: %s", e.Field, e.Message)
}

func validateAge(age int) error {
	if age < 0 {
		return &ValidationError{Field: "Age", Message: "must not be negative"}
	}
	return nil
}

// -----------------------------------------------------------------------
// 4. Builder pattern using method chaining (fluent API)
// -----------------------------------------------------------------------

type Pizza struct {
	Size     string
	Toppings []string
	Cheese   bool
}

type PizzaBuilder struct {
	pizza Pizza
}

func NewPizzaBuilder() *PizzaBuilder {
	return &PizzaBuilder{}
}

func (b *PizzaBuilder) Size(size string) *PizzaBuilder {
	b.pizza.Size = size
	return b
}

func (b *PizzaBuilder) AddTopping(t string) *PizzaBuilder {
	b.pizza.Toppings = append(b.pizza.Toppings, t)
	return b
}

func (b *PizzaBuilder) WithCheese() *PizzaBuilder {
	b.pizza.Cheese = true
	return b
}

func (b *PizzaBuilder) Build() Pizza {
	return b.pizza
}

// -----------------------------------------------------------------------
// 5. Functional options pattern (idiomatic Go alternative to builders)
// -----------------------------------------------------------------------

type Server struct {
	Host    string
	Port    int
	Timeout int
}

type ServerOption func(*Server)

func WithHost(host string) ServerOption {
	return func(s *Server) { s.Host = host }
}

func WithPort(port int) ServerOption {
	return func(s *Server) { s.Port = port }
}

func WithTimeout(timeout int) ServerOption {
	return func(s *Server) { s.Timeout = timeout }
}

func NewServer(opts ...ServerOption) *Server {
	s := &Server{Host: "localhost", Port: 8080, Timeout: 30} // defaults
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// -----------------------------------------------------------------------
// 6. Sorting a slice of structs with sort.Slice
// -----------------------------------------------------------------------

type Employee struct {
	Name   string
	Salary int
}

// -----------------------------------------------------------------------
// 7. Unmarshalling JSON into a slice of structs
// -----------------------------------------------------------------------

type Book struct {
	Title  string `json:"title"`
	Author string `json:"author"`
	Year   int    `json:"year"`
}

// -----------------------------------------------------------------------
// 8. Struct embedding an interface (not just a struct)
// -----------------------------------------------------------------------

type Logger interface {
	Log(msg string)
}

type ConsoleLogger struct{}

func (ConsoleLogger) Log(msg string) { fmt.Println("[LOG]", msg) }

type Service struct {
	Logger // embedded interface -> Service gets a Log method for free
	Name   string
}

// -----------------------------------------------------------------------
// 9. Deep vs shallow copy pitfalls (slice/map fields inside structs)
// -----------------------------------------------------------------------

type Team struct {
	Name    string
	Members []string
}

func (t Team) ShallowCopy() Team {
	return t // struct itself copied, but Members slice header points to same array
}

func (t Team) DeepCopy() Team {
	membersCopy := make([]string, len(t.Members))
	copy(membersCopy, t.Members)
	return Team{Name: t.Name, Members: membersCopy}
}

// -----------------------------------------------------------------------
// main
// -----------------------------------------------------------------------

func main() {
	fmt.Println("=== 1. Structs implementing an interface ===")
	shapes := []Shape{Circle{Radius: 3}, Square{Side: 4}}
	for _, s := range shapes {
		describe(s)
	}
	fmt.Println()

	fmt.Println("=== 2. Custom String() method ===")
	price := Money{Cents: 1999}
	fmt.Println("  price:", price) // uses String() automatically
	fmt.Println()

	fmt.Println("=== 3. Custom error struct ===")
	if err := validateAge(-5); err != nil {
		fmt.Println("  error:", err)
		var ve *ValidationError
		if errors.As(err, &ve) {
			fmt.Println("  field that failed:", ve.Field)
		}
	}
	fmt.Println()

	fmt.Println("=== 4. Builder pattern ===")
	pizza := NewPizzaBuilder().
		Size("Large").
		AddTopping("Mushroom").
		AddTopping("Olives").
		WithCheese().
		Build()
	fmt.Printf("  pizza=%+v\n\n", pizza)

	fmt.Println("=== 5. Functional options pattern ===")
	srv := NewServer(WithHost("0.0.0.0"), WithPort(9090))
	fmt.Printf("  server=%+v (Timeout kept default)\n\n", srv)

	fmt.Println("=== 6. Sorting a slice of structs ===")
	employees := []Employee{
		{"Alice", 70000},
		{"Bob", 55000},
		{"Charlie", 90000},
	}
	sort.Slice(employees, func(i, j int) bool {
		return employees[i].Salary > employees[j].Salary // descending
	})
	for _, e := range employees {
		fmt.Printf("  %-8s $%d\n", e.Name, e.Salary)
	}
	fmt.Println()

	fmt.Println("=== 7. Unmarshalling JSON into structs ===")
	jsonData := `[
		{"title":"The Go Programming Language","author":"Donovan & Kernighan","year":2015},
		{"title":"Clean Code","author":"Robert Martin","year":2008}
	]`
	var books []Book
	if err := json.Unmarshal([]byte(jsonData), &books); err != nil {
		fmt.Println("  unmarshal error:", err)
	}
	for _, b := range books {
		fmt.Printf("  %s by %s (%d)\n", b.Title, b.Author, b.Year)
	}
	fmt.Println()

	fmt.Println("=== 8. Embedded interface ===")
	svc := Service{Logger: ConsoleLogger{}, Name: "AuthService"}
	svc.Log("service started") // promoted from embedded Logger interface
	fmt.Println()

	fmt.Println("=== 9. Shallow vs deep copy pitfall ===")
	original := Team{Name: "Alpha", Members: []string{"Sam", "Priya"}}
	shallow := original.ShallowCopy()
	deep := original.DeepCopy()

	shallow.Members[0] = "MUTATED"     // affects original too! (shared backing array)
	deep.Members[1] = "SAFE-CHANGE"    // does NOT affect original

	fmt.Println("  original.Members:", original.Members, "<- mutated via shallow copy")
	fmt.Println("  shallow.Members: ", shallow.Members)
	fmt.Println("  deep.Members:    ", deep.Members)
}