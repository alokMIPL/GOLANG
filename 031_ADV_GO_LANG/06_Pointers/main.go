package main

import "fmt"

func main() {

	// Store the memory address of any val

	// &x -> address of x (makes a pointer)

	// *p -> dereferencing (go to that address and read.write)

	// We use pointer because we want to change any value in function and don't want to return it.

	score := 10

	fmt.Println("before:", score)

	addScore(&score)

	fmt.Println("after:", score)

	/*
		Output :
		before: 10
		after: 15
	*/

}

func addScore(score *int) {

	*score = *score + 5

}
