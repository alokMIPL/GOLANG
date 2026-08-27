package main

import "fmt"

// We pass it by value
// In pass by value we copt the value of num to another varaible not chnage the actual variale value.
func changeNum(num int) {
	num = 5
	fmt.Println("In changeNum", num)
}

/*
main()'s stack frame:
┌─────────────┐
│ num = 1     │  ← address, say 0xA
└─────────────┘

changeNum(num) is called → Go copies the VALUE (1) into a new variable
                                    │
                                    ▼
changeNum()'s stack frame:
┌─────────────┐
│ num = 1     │  ← address, say 0xB (a DIFFERENT memory location)
└─────────────┘
        │
        ▼
num = 5   →  only 0xB changes
┌─────────────┐
│ num = 5     │  ← 0xB
└─────────────┘

Back in main(): 0xA was never touched, so it's still 1

*/

// Now pass by reference this means "POINTERS"

// By reference

func changeNumRef(num *int) {
	*num = 5
	fmt.Println("In changeNumRef", *num)
}

func main() {

	num := 1

	fmt.Println("pass by Value")
	changeNum(num)
	fmt.Println("After changeNum in main", num)

	fmt.Println("**************")

	fmt.Println("pass by Reference")
	changeNumRef(&num)
	fmt.Println("After changeNumRef in main", num)

}
