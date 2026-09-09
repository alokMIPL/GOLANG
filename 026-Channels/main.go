package main

import (
	"fmt"
	"time"
)

// When we have multiple goroutines running concurrently, we need a way to communicate between them. Channels provide a way for one goroutine to send data to another goroutine. Channels are a powerful feature of Go that allow us to build concurrent programs that are easy to reason about.

// DEADLOCK
// A deadlock is a situation where a program (or a set of goroutines/threads) gets stuck forever, because everyone involved is waiting on something that will never happen.

// 2. Basic Channel
func processNum(num chan int) {
	fmt.Println("Processing Number", <-num)
}

// 3. Sending Data to Channels
func processNumLoop(numChan chan int) {
	for num := range numChan {
		fmt.Println("Processing Number", num)
	}
}

// 4. Here we receive data from function to CHANNEL
func sum(result chan int, num1 int, num2 int) {
	numResult := num1 + num2
	result <- numResult
}

// 5. We can do waitGroup functionality by using CHANNEL also.
func task(done chan bool) {

	// Here we use difer function, it run when all the take is completed. If function give error then also it run.
	defer func() { done <- true }()
	fmt.Println("Processing...")
}

// 6. Email sender by CHANNEL

func emailSender(emailChan chan string, emailDone chan bool) {
	defer func() { emailDone <- true }()
	for email := range emailChan {
		fmt.Println("Sending email to", email)
		time.Sleep(time.Second)
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

	// numChan := make(chan int)
	// go processNumLoop(numChan)

	// We sent random number to channel, So we use loop and rand function.

	// for {
	// 	numChan <- rand.Intn(100)
	// }

	// 4. Here we receive data from function to CHANNEL. ************

	// result := make(chan int)
	// go sum(result, 4, 5)
	// res := <-result
	// fmt.Println(res)

	// 5. We can do waitGroup functionality by using CHANNEL also. ************

	done := make(chan bool)
	go task(done)
	<-done
	// here program comes and block. Basically here function ends.

	// Now 6. ************
	// In earlier CHANNEL form 1 to 5, the condition is when we only send one data at a time, and wheneve the data not received then we not able to send the new data into that CHANNEL.
	// So there we called NON-BUFFER CHANNEL

	// Now Buffer CHANNEL
	// In Buffer channel we can send limited amoun of data without blocking.

	// Suppose we can create a channel for email system

	emailChan := make(chan string, 100)
	emailDone := make(chan bool)

	go emailSender(emailChan, emailDone)
	/*
		Here we write 100 that means in buffer we have a space of sending 100 items.
		So, up to 100 this will work and no DEADLOCK happen. But if we increase the value form more than 100 then it start the DEADLOCK.
	*/

	// we dont generate our email like this
	// emailChan <- "1@example.com"
	// emailChan <- "2@example.com"
	// fmt.Println(<-emailChan)
	// fmt.Println(<-emailChan)

	for i := 0; i < 100; i++ {
		emailChan <- fmt.Sprintf("%d@gmail.com", i)
	}

	fmt.Println("done sending...")

	<-emailDone

}
