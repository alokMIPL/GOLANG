package mathutils

import "fmt"

const Pi = 3.14159

const precisionDigits = 2

func init() {
	fmt.Println("[mathutils] package initialized")
}

func Add(a, b int) int {
	return a + b
}

func Square(a int) int {
	return a * a
}

func CircleArea(radius float64) float64 {
	return Pi * radius * radius
}

func average(nums []int) float64 {
	sum := 0
	for _, n := range nums {
		sum += n
	}
	return float64(sum) / float64(len(nums))
}

func Average(nums []int) float64 {
	return average(nums)
}
