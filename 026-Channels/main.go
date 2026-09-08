package main

import "fmt"

// When we have multiple goroutines running concurrently, we need a way to communicate between them. Channels provide a way for one goroutine to send data to another goroutine. Channels are a powerful feature of Go that allow us to build concurrent programs that are easy to reason about.

// DEADLOCK
// A deadlock is a situation where a program (or a set of goroutines/threads) gets stuck forever, because everyone involved is waiting on something that will never happen.

func main() {

	// How to create a channel in Go?
	messageChan := make(chan string)

	// How to send a message to a channel in Go?
	messageChan <- "Ping, channel!"

	// How to receive a message from a channel in Go?
	msg := <-messageChan
	fmt.Println(msg)

}
