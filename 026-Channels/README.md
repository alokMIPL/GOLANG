# Go Channels — Complete Guide & Code Walkthrough

This README explains, in detail, the concepts and code in `main.go`, which demonstrates how **channels** work in Go for communication between goroutines.

---

## Table of Contents

1. [What Are Channels?](#what-are-channels)
2. [What Is a Deadlock?](#what-is-a-deadlock)
3. [Section 1: Deadlock Example](#section-1-deadlock-example)
4. [Section 2: Basic Channel](#section-2-basic-channel)
5. [Section 3: Sending Data with a Loop (`range`)](#section-3-sending-data-with-a-loop-range)
6. [Section 4: Returning Data via Channel from a Function](#section-4-returning-data-via-channel-from-a-function)
7. [Section 5: WaitGroup-like Behavior Using Channels](#section-5-waitgroup-like-behavior-using-channels)
8. [Section 6: Buffered Channels (Email Sender Example)](#section-6-buffered-channels-email-sender-example)
9. [Section 7: Receiving from Multiple Channels (`select`)](#section-7-receiving-from-multiple-channels-select)
10. [How to Run](#how-to-run)
11. [Key Takeaways](#key-takeaways)

---

## What Are Channels?

Channels are Go's built-in mechanism for goroutines to **communicate and synchronize** with each other. Instead of sharing memory and protecting it with locks, Go encourages the philosophy:

> "Do not communicate by sharing memory; instead, share memory by communicating."

A channel is created with `make(chan Type)` and supports two core operations:

| Operation | Syntax | Meaning |
|---|---|---|
| Send | `channel <- value` | Sends `value` into the channel |
| Receive | `value := <-channel` | Receives a value from the channel |

By default, channels are **unbuffered** — a send blocks until another goroutine is ready to receive, and vice versa.

---

## What Is a Deadlock?

A **deadlock** happens when a goroutine (or the whole program) waits forever for something that will never occur — usually because no other goroutine will ever perform the matching send/receive operation.

Go's runtime can actually detect this in simple cases and will crash with:
```
fatal error: all goroutines are asleep - deadlock!
```

---

## Section 1: Deadlock Example

```go
// messageChan := make(chan string)
// messageChan <- "Ping, channel!"
// msg := <-messageChan
// fmt.Println(msg)
```

This block is commented out intentionally because it **would deadlock** if run as-is:

1. `messageChan` is an **unbuffered channel** — it has no storage capacity.
2. `messageChan <- "Ping, channel!"` tries to send a value on the **main goroutine**.
3. Since there is no other goroutine concurrently trying to receive from `messageChan`, the send blocks forever.
4. Nobody else is running to unblock it → **deadlock**.

**Lesson:** For an unbuffered channel to work, you need a send and a receive happening in *different* goroutines at (roughly) the same time.

---

## Section 2: Basic Channel
