package main

import (
	"fmt"
	"math/rand"
	"time"
)

// When we have multiple goroutines running concurrently, we need a way to communicate between them. Channels provide a way for one goroutine to send data to another goroutine. Channels are a powerful feature of Go that allow us to build concurrent programs that are easy to reason about.

// DEADLOCK
// A deadlock is a situation where a program (or a set of goroutines/threads) gets stuck forever, because everyone involved is waiting on something that will never happen.

func processNum(num chan int) {
	fmt.Println("Processing Number", <-num)
}

func processNumLoop(numChan chan int) {
	for num := range numChan {
		fmt.Println("Processing Number", num)
	}
}

func main() {

	// 1. DeadLock Example ************
	// How to create a channel in Go?

	// messageChan := make(chan string)

	// How to send a message to a channel in Go?

	// messageChan <- "Ping, channel!"

	// now this will cause a deadlock because the main goroutine is trying to send a message to the channel, but there is no other goroutine that is receiving from the channel. So, the main goroutine will be blocked forever.

	// How to receive a message from a channel in Go?

	// msg := <-messageChan
	// fmt.Println(msg)

	// 2. Channel Example by using a function to process the number. ************
	num := make(chan int)

	go processNum(num)
	num <- 5
	time.Sleep(time.Second * 2)

	// 3. Channel Example by using a function to process the number. ************
	numChan := make(chan int)

	go processNumLoop(numChan)

	// We sent random number to channel, So we use loop and ran function.
	for {
		numChan <- rand.Intn(100)
	}

}
