// The main package is special: it's the entry point of an executable
// program (as opposed to a "library" package like mathutils above).
// Putting it under cmd/app/ is a common Go project-layout convention
// when a module produces one or more binaries.
package main

import (
	"fmt"

	m "go-packages-demo/mathutils" // import alias: m.Add(...) instead of mathutils.Add(...)
	"go-packages-demo/shapes"
	"go-packages-demo/stringutils"

	"go-packages-demo/internal/config" // allowed: main is inside the same module
)

func main() {
	fmt.Println("=== Using mathutils (aliased as m) ===")
	fmt.Println("  m.Add(2, 3) =", m.Add(2, 3))
	fmt.Println("  m.Square(5) =", m.Square(5))
	fmt.Printf("  m.CircleArea(2) = %.2f (using m.Pi = %.5f)\n", m.CircleArea(2), m.Pi)
	fmt.Println("  m.Average([1,2,3,4]) =", m.Average([]int{1, 2, 3, 4}))
	fmt.Println()

	fmt.Println("=== Using stringutils (multi-file package) ===")
	fmt.Println("  Reverse(\"hello\") =", stringutils.Reverse("hello"))
	fmt.Println("  IsPalindrome(\"A man a plan a canal Panama\") =",
		stringutils.IsPalindrome("A man a plan a canal Panama"))
	fmt.Println("  CountVowels(\"Golang Packages\") =", stringutils.CountVowels("Golang Packages"))
	fmt.Println("  Title(\"go PACKAGES demo\") =", stringutils.Title("go PACKAGES demo"))
	fmt.Println()

	fmt.Println("=== Using shapes (interface + structs across a package) ===")
	c := shapes.Circle{Radius: 3}
	r := shapes.Rectangle{Width: 4, Height: 5}
	fmt.Printf("  %s area = %.2f\n", c.Name(), c.Area())
	fmt.Printf("  %s area = %.2f\n", r.Name(), r.Area())
	fmt.Printf("  TotalArea(c, r) = %.2f\n", shapes.TotalArea(c, r))
	fmt.Println()

	fmt.Println("=== Using internal/config (only importable within this module) ===")
	cfg := config.Load()
	fmt.Printf("  cfg = %+v\n", cfg)
}
