package mathutils

import (
	"fmt"
	"testing"
)

func TestAdd(t *testing.T) {
	cases := []struct {
		name     string
		a, b     int
		expected int
	}{
		{"positives", 2, 3, 5},
		{"with negative", -2, 5, 3},
		{"zeros", 0, 0, 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Add(tc.a, tc.b)
			if got != tc.expected {
				t.Errorf("Add(%d, %d) = %d; want %d", tc.a, tc.b, got, tc.expected)
			}
		})
	}
}

func TestAverage(t *testing.T) {
	got := Average([]int{2, 4, 6})
	want := 4.0
	if got != want {
		t.Errorf("Average() = %v; want %v", got, want)
	}
}

func BenchmarkSquare(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Square(42)
	}
}

func ExampleAdd() {
	sum := Add(3, 4)
	fmt.Println(sum)
	// Output: 7
}

func ExampleCircleArea() {
	fmt.Printf("%.2f\n", CircleArea(1))
	// Output: 3.14
}
