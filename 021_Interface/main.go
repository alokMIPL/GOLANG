// structs_reflection.go
// Part 4: reflection, multi-interface embedding, DB-row scanning, and struct validation.
// Run with: go run structs_reflection.go

package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

// -----------------------------------------------------------------------
// 1. Struct embedding MULTIPLE interfaces at once
// -----------------------------------------------------------------------

type Reader interface {
	Read() string
}

type Writer interface {
	Write(s string)
}

type FileHandle struct {
	Reader
	Writer
	Name string
}

type memReader struct{ data string }

func (m memReader) Read() string { return m.data }

type memWriter struct{ buf *string }

func (m memWriter) Write(s string) { *m.buf = *m.buf + s }

// -----------------------------------------------------------------------
// 2. Reflection: inspect struct fields, types, and tags at runtime
// -----------------------------------------------------------------------

type Product struct {
	Name  string  `json:"name" validate:"required"`
	Price float64 `json:"price" validate:"gt=0"`
	SKU   string  `json:"sku" validate:"required"`
}

func inspect(v interface{}) {
	t := reflect.TypeOf(v)
	val := reflect.ValueOf(v)
	fmt.Printf("  Type: %s\n", t.Name())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		fmt.Printf("    Field: %-6s Type: %-8s Value: %-10v json:%-10q validate:%q\n",
			field.Name,
			field.Type,
			val.Field(i).Interface(),
			field.Tag.Get("json"),
			field.Tag.Get("validate"),
		)
	}
}

// -----------------------------------------------------------------------
// 3. Minimal reflection-based validator using struct tags
// -----------------------------------------------------------------------

func validate(v interface{}) []string {
	var errs []string
	t := reflect.TypeOf(v)
	val := reflect.ValueOf(v)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		rule := field.Tag.Get("validate")
		if rule == "" {
			continue
		}
		fieldVal := val.Field(i)

		switch rule {
		case "required":
			if fieldVal.Kind() == reflect.String && fieldVal.String() == "" {
				errs = append(errs, fmt.Sprintf("%s is required", field.Name))
			}
		case "gt=0":
			if fieldVal.Kind() == reflect.Float64 && fieldVal.Float() <= 0 {
				errs = append(errs, fmt.Sprintf("%s must be greater than 0", field.Name))
			}
		}
	}
	return errs
}

// -----------------------------------------------------------------------
// 4. Scanning database rows into a struct (database/sql pattern)
// -----------------------------------------------------------------------

type Customer struct {
	ID    int
	Name  string
	Email string
}

// scanCustomer shows the standard idiom for mapping a sql.Rows row -> struct.
func scanCustomer(rows *sql.Rows) (Customer, error) {
	var c Customer
	err := rows.Scan(&c.ID, &c.Name, &c.Email)
	return c, err
}

// -----------------------------------------------------------------------
// 5. Struct implementing sql.Scanner + driver.Valuer (custom DB type)
// -----------------------------------------------------------------------

type Status int

const (
	StatusActive Status = iota
	StatusInactive
)

func (s Status) String() string {
	if s == StatusActive {
		return "active"
	}
	return "inactive"
}

// Scan implements sql.Scanner so *Status can be passed to rows.Scan directly.
func (s *Status) Scan(value interface{}) error {
	str, ok := value.(string)
	if !ok {
		return fmt.Errorf("cannot scan %T into Status", value)
	}
	if str == "active" {
		*s = StatusActive
	} else {
		*s = StatusInactive
	}
	return nil
}

// -----------------------------------------------------------------------
// 6. Struct diffing: compare two structs field by field via reflection
// -----------------------------------------------------------------------

func diff(a, b interface{}) map[string][2]interface{} {
	changes := make(map[string][2]interface{})
	ta, va := reflect.TypeOf(a), reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	for i := 0; i < ta.NumField(); i++ {
		name := ta.Field(i).Name
		oldVal := va.Field(i).Interface()
		newVal := vb.Field(i).Interface()
		if !reflect.DeepEqual(oldVal, newVal) {
			changes[name] = [2]interface{}{oldVal, newVal}
		}
	}
	return changes
}

// -----------------------------------------------------------------------
// 7. Generic "Set" of struct fields via a fluent updater (patch pattern)
// -----------------------------------------------------------------------

type UserProfile struct {
	Name string
	Bio  string
	Age  int
}

type UserProfilePatch struct {
	Name *string
	Bio  *string
	Age  *int
}

func (p UserProfile) Apply(patch UserProfilePatch) UserProfile {
	if patch.Name != nil {
		p.Name = *patch.Name
	}
	if patch.Bio != nil {
		p.Bio = *patch.Bio
	}
	if patch.Age != nil {
		p.Age = *patch.Age
	}
	return p
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

// -----------------------------------------------------------------------
// 8. Struct as a simple event/message with type switch dispatch
// -----------------------------------------------------------------------

type UserCreated struct{ Name string }
type UserDeleted struct{ ID int }

func handleEvent(e interface{}) {
	switch ev := e.(type) {
	case UserCreated:
		fmt.Printf("  [event] user created: %s\n", ev.Name)
	case UserDeleted:
		fmt.Printf("  [event] user deleted: id=%d\n", ev.ID)
	default:
		fmt.Println("  [event] unknown event type")
	}
}

// -----------------------------------------------------------------------
// main
// -----------------------------------------------------------------------

func main() {
	fmt.Println("=== 1. Struct embedding multiple interfaces ===")
	var buf string
	fh := FileHandle{
		Reader: memReader{data: "hello from file"},
		Writer: memWriter{buf: &buf},
		Name:   "notes.txt",
	}
	fmt.Println("  fh.Read():", fh.Read()) // promoted from Reader
	fh.Write("appended text")              // promoted from Writer
	fmt.Println("  buf after Write:", buf)
	fmt.Println()

	fmt.Println("=== 2. Reflection: inspecting struct fields + tags ===")
	p := Product{Name: "Widget", Price: 9.99, SKU: "W-001"}
	inspect(p)
	fmt.Println()

	fmt.Println("=== 3. Reflection-based validation ===")
	bad := Product{Name: "", Price: -5, SKU: "X-1"}
	if errs := validate(bad); len(errs) > 0 {
		fmt.Println("  validation errors:")
		for _, e := range errs {
			fmt.Println("   -", e)
		}
	}
	fmt.Println()

	fmt.Println("=== 4 & 5. database/sql struct scanning (illustrative, no real DB) ===")
	fmt.Println("  scanCustomer(rows) would populate Customer{ID, Name, Email} via rows.Scan")
	var st Status
	_ = st.Scan("active")
	fmt.Println("  Status after Scan(\"active\"):", st)
	fmt.Println()

	fmt.Println("=== 6. Struct diffing via reflection ===")
	before := UserProfile{Name: "Alice", Bio: "old bio", Age: 30}
	after := UserProfile{Name: "Alice", Bio: "new bio", Age: 31}
	changes := diff(before, after)
	for field, vals := range changes {
		fmt.Printf("  %s changed: %v -> %v\n", field, vals[0], vals[1])
	}
	fmt.Println()

	fmt.Println("=== 7. Patch/partial-update pattern with pointer fields ===")
	updated := before.Apply(UserProfilePatch{Bio: strPtr("updated bio"), Age: intPtr(32)})
	fmt.Printf("  before=%+v\n  updated=%+v\n\n", before, updated)

	fmt.Println("=== 8. Struct-based events with type switch ===")
	events := []interface{}{
		UserCreated{Name: "Dev"},
		UserDeleted{ID: 42},
		"not an event",
	}
	for _, e := range events {
		handleEvent(e)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("-", 40))
	fmt.Println("Done.")
}
