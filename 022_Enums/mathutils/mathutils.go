// Package mathutils provides small numeric helper functions.
//
// This comment directly above "package mathutils" is the package doc comment
// — it shows up when someone runs `go doc mathutils` or views pkg.go.dev.
package mathutils

import "fmt"

// Pi is an exported constant — capitalized identifiers are visible
// to any package that imports mathutils.
const Pi = 3.14159

// precisionDigits is unexported — only visible inside this package.
const precisionDigits = 2

// init() runs automatically once, before main(), the first time this
// package is imported anywhere in the program. Useful for one-time setup.
func init() {
	fmt.Println("[mathutils] package initialized")
}

// Add returns the sum of two integers. Exported (capitalized) -> usable
// by other packages as mathutils.Add(...).
func Add(a, b int) int {
	return a + b
}

// Square returns a squared.
func Square(a int) int {
	return a * a
}

// CircleArea uses the exported Pi constant.
func CircleArea(radius float64) float64 {
	return Pi * radius * radius
}

// average is unexported — a private helper only mathutils.go (or other
// files in package mathutils) can call directly.
func average(nums []int) float64 {
	sum := 0
	for _, n := range nums {
		sum += n
	}
	return float64(sum) / float64(len(nums))
}

// Average is the exported wrapper around the private average() helper.
// This is a common pattern: expose a clean public API while keeping
// implementation details private.
func Average(nums []int) float64 {
	return average(nums)
}
