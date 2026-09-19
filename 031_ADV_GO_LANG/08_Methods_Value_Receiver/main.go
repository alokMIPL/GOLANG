package main

import "fmt"

type User struct {
	Name string
	Age  int
}

func main() {

	u := User{Name: "Alok", Age: 24}
	fmt.Println(u.Intro())

}

func (u User) Intro() string {
	return fmt.Sprintf("Hi, I am %s", u.Name)
}
