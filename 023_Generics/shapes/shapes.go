// Package shapes defines geometric shapes and a common Shape interface.
package shapes

import "math"

// Shape is the exported interface every shape in this package satisfies.
type Shape interface {
	Area() float64
	Name() string
}

// Circle is an exported struct.
type Circle struct {
	Radius float64
}

func (c Circle) Area() float64 { return math.Pi * c.Radius * c.Radius }
func (c Circle) Name() string  { return "Circle" }

// Rectangle is an exported struct.
type Rectangle struct {
	Width, Height float64
}

func (r Rectangle) Area() float64 { return r.Width * r.Height }
func (r Rectangle) Name() string  { return "Rectangle" }

// TotalArea sums the area of any number of shapes — demonstrates
// a package function operating on its own interface type.
func TotalArea(shapes ...Shape) float64 {
	total := 0.0
	for _, s := range shapes {
		total += s.Area()
	}
	return total
}
