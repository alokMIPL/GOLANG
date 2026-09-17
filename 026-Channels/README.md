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

```go
func processNum(num chan int) {
	fmt.Println("Processing Number", <-num)
}
```

```go
num := make(chan int)
go processNum(num)
num <- 5
time.Sleep(time.Second * 2)
```

**What's happening:**
- `num` is created as an unbuffered `chan int`.
- `go processNum(num)` starts `processNum` as a **new goroutine**, which immediately tries to receive (`<-num`) and blocks, waiting for a value.
- Back in `main`, `num <- 5` sends the value `5`. Since `processNum` is already waiting to receive, this send succeeds immediately and both goroutines "sync up" at that point.
- `processNum` then prints `Processing Number 5`.
- `time.Sleep(time.Second * 2)` is used here just to **give the goroutine time to finish and print** before `main()` moves on (a manual — and fragile — substitute for proper synchronization, which is improved on in later sections using channels/WaitGroups).

---

## Section 3: Sending Data with a Loop (`range`)

```go
func processNumLoop(numChan chan int) {
	for num := range numChan {
		fmt.Println("Processing Number", num)
	}
}
```

This function shows how to **continuously consume values** from a channel using `for range`:

- `for num := range numChan` keeps receiving values from `numChan` **until the channel is closed**.
- Once the channel is closed (and drained of any remaining buffered values), the loop exits automatically.
- This pattern is common for worker goroutines that process a stream of values.

The example usage (also commented out) shows a producer sending random numbers in an infinite loop:
```go
// for {
//     numChan <- rand.Intn(100)
// }
```
This is left commented out because it runs forever and would need a `close(numChan)` or a break condition to terminate cleanly.

---

## Section 4: Returning Data via Channel from a Function

```go
func sum(result chan int, num1 int, num2 int) {
	numResult := num1 + num2
	result <- numResult
}
```

Usage (commented out):
```go
// result := make(chan int)
// go sum(result, 4, 5)
// res := <-result
// fmt.Println(res)
```

**Purpose:** Since a goroutine started with `go` cannot directly `return` a value to the caller, channels are used to **pass the result back**.

- `sum` computes `num1 + num2` and sends the result into the `result` channel.
- The caller (`main`) blocks on `res := <-result` until the goroutine finishes and sends the sum.
- This is a fundamental pattern: **channels as a return mechanism for concurrent functions**.

---

## Section 5: WaitGroup-like Behavior Using Channels

```go
func task(done chan bool) {
	defer func() { done <- true }()
	fmt.Println("Processing...")
}
```

```go
done := make(chan bool)
go task(done)
<-done
```

**What's happening:**
- `task` prints `"Processing..."`, and uses `defer` to guarantee that `done <- true` runs **after** the function body finishes — even if an error/panic occurred, since deferred functions still execute.
- In `main`, `<-done` blocks until the `task` goroutine signals completion by sending `true`.
- This achieves the same effect as `sync.WaitGroup.Wait()` — **the main goroutine waits for a background goroutine to finish** — but implemented manually with a plain channel used purely as a "signal", not for its data.

This is a very common Go idiom: using `chan bool` or `chan struct{}` purely as a **completion signal**, where the value itself doesn't matter.

---

## Section 6: Buffered Channels (Email Sender Example)

### Buffered vs Unbuffered Channels

| Type | Declaration | Behavior |
|---|---|---|
| Unbuffered | `make(chan T)` | Send blocks until a receiver is ready |
| Buffered | `make(chan T, N)` | Send only blocks once the buffer of size `N` is full |

```go
func emailSender(emailChan chan string, emailDone chan bool) {
	defer func() { emailDone <- true }()
	for email := range emailChan {
		fmt.Println("Sending email to", email)
		time.Sleep(time.Second)
	}
}
```

```go
emailChan := make(chan string, 100)
emailDone := make(chan bool)

go emailSender(emailChan, emailDone)

for i := 0; i < 5; i++ {
	emailChan <- fmt.Sprintf("%d@gmail.com", i)
}

fmt.Println("done sending...")
close(emailChan)
<-emailDone
```

**Step-by-step:**
1. `emailChan` is a **buffered channel** with capacity `100` — up to 100 emails can be queued without blocking the sender.
2. `emailDone` is an unbuffered signal channel, just like in Section 5.
3. `emailSender` runs as a goroutine, looping over `emailChan` with `range`, printing and "sending" each email (simulated via `time.Sleep(time.Second)`).
4. In `main`, a loop sends 5 emails into `emailChan`. Because the buffer size (100) is much larger than 5, **none of these sends block** — `main` can queue all 5 emails almost instantly.
5. `close(emailChan)` signals that **no more values will be sent**. This is essential — without it, the `for range` loop inside `emailSender` would wait forever for more values, causing a deadlock.
6. `<-emailDone` blocks `main` until `emailSender` has processed all queued emails and finished (signaled via the deferred `emailDone <- true`).

**Important note from the code comment:** If you try to send **more items than the buffer capacity** without a concurrent receiver draining it, the extra sends will block, and if nothing is receiving, this results in a deadlock. E.g., sending 101+ emails into a channel of capacity 100 with no consumer running would eventually block forever.

**Best practice reminder:** Only the **sender** should `close()` a channel, and only when it's certain no more values will be sent. Sending on a closed channel causes a panic; receiving from a closed (and drained) channel returns the zero value immediately without blocking.

---

## Section 7: Receiving from Multiple Channels (`select`)

```go
chan1 := make(chan int)
chan2 := make(chan string)

go func() {
	chan1 <- 10
}()

go func() {
	chan2 <- "GOLANG"
}()

for i := 0; i < 2; i++ {
	select {
	case chan1Val := <-chan1:
		fmt.Println("Received Data from chan1", chan1Val)
	case chan2Val := <-chan2:
		fmt.Println("Received Data from chan2", chan2Val)
	}
}
```

**What's happening:**
- Two unbuffered channels (`chan1` — int, `chan2` — string) are created.
- Two anonymous goroutines are launched, each sending a value on one of the channels.
- The `select` statement lets `main` **wait on multiple channel operations at once**. It blocks until **one** of its `case`s is ready, then executes that case.
- The `for i := 0; i < 2; i++` loop runs `select` twice, so that **both** values (from `chan1` and `chan2`) get received — since we don't know which goroutine will be ready first, `select` handles whichever is ready and the loop ensures we don't miss the other one.
- The **order of output is not guaranteed** — either "Received Data from chan1 10" or "Received Data from chan2 GOLANG" could print first, depending on which goroutine's send is scheduled first by the Go runtime.

**Why `select` matters:** It's the mechanism Go provides for **non-deterministic, multi-way channel communication** — essential for building things like fan-in patterns, timeouts (`select` with `time.After`), and cancellation (`select` with a `context.Done()` channel).

---

## How to Run

```bash
go run main.go
```

Expected behavior:
- Prints a message from `processNum`.
- Prints `"Processing..."`.
- Prints `"Sending email to X@gmail.com"` five times, one second apart, followed by `"done sending..."` (note: due to goroutine scheduling, `"done sending..."` may print before all emails are logged, since sending to the buffer doesn't wait for the emails to actually be "sent").
- Prints two lines for `chan1`/`chan2`, in a non-deterministic order.

---

## Key Takeaways

- **Unbuffered channels** synchronize sender and receiver — a send only completes when a receive is ready (and vice versa).
- **Buffered channels** allow a fixed number of values to be queued without an immediate receiver, but will still block once the buffer is full.
- **`close(channel)`** should be called by the sender to signal "no more data," allowing `for range` loops on the channel to terminate.
- **Channels can act as return values** for concurrent functions (Section 4) or as pure **completion signals** (Section 5), similar to `sync.WaitGroup`.
- **`select`** lets a goroutine wait on multiple channel operations simultaneously and proceeds with whichever is ready first.
- **Deadlocks** occur when goroutines wait on operations that can never complete — commonly from unbuffered channels with no matching send/receive, unclosed channels being ranged over, or buffer capacity being exceeded with no consumer.
