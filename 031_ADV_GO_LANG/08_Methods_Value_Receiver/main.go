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

/*
You're mostly right, with two small corrections.

What's right
Intro is a method.
User is the struct type.
The user is handed to the method, so Intro can use its data.
Correction 1: the receiver is not a normal input

(u User) is called the receiver. It is not listed with the normal inputs. It is the value you call the method on:

go
u.Intro()   // u is passed in as the receiver

A regular function would take the user as an argument:

go
func Intro(u User) string { ... }
Intro(u)

The method version has the same effect. Go just lets you write u.Intro() instead.

Correction 2: it doesn't return u.Name

It returns a new string, with the name inserted into it:

go
return fmt.Sprintf("Hi, I am %s", u.Name)
Code	Result
u.Name	"Alok"
What Intro returns	"Hi, I am Alok"

%s is a placeholder that Sprintf replaces with u.Name.

"Method value" is also a specific Go term. It means writing u.Intro without the (), which gives you the method itself as a function. What Intro returns is just a string.

Summary
go
func (u User) Intro() string {
//   ^^^^^^^^ receiver: the user the method is called on
//            ^^^^^ method name
//                    ^^^^^^ return type: text
	return fmt.Sprintf("Hi, I am %s", u.Name)
}

Intro is a method on User that takes the user's name and returns the text "Hi, I am <name>".
*/
