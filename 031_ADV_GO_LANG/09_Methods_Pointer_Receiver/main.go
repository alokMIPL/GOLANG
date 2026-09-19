package main

import "fmt"

type User struct {
	Name string
	Age  int
}

func main() {

	u := User{Name: "Alok", Age: 24}
	fmt.Println("Before Methods Pointer Receiver", u.Age)

	u.Birthday()

	fmt.Println("After Methods Pointer Receiver", u.Age)

}

func (u *User) Birthday() {
	u.Age++
}

/*
Output =
Before Methods Pointer Receiver 24
After Methods Pointer Receiver 25
*/
