# Go Mutex & Race Conditions — Complete Guide & Code Walkthrough

This README explains, in detail, the concepts and code in `main.go`, which demonstrates how to use `sync.Mutex` in Go to prevent **race conditions** when multiple goroutines modify a shared resource.

---

## Table of Contents

1. [What Is a Race Condition?](#what-is-a-race-condition)
2. [What Is a Mutex?](#what-is-a-mutex)
3. [The `post` Struct](#the-post-struct)
4. [The `inc` Method — Locking a Shared Resource](#the-inc-method--locking-a-shared-resource)
5. [Why `Unlock()` Is Called Inside `defer`](#why-unlock-is-called-inside-defer)
6. [The `main` Function — Spawning 100 Goroutines](#the-main-function--spawning-100-goroutines)
7. [What Happens Without a Mutex (Observed Behavior)](#what-happens-without-a-mutex-observed-behavior)
8. [Mutex Best Practices Highlighted in the Code](#mutex-best-practices-highlighted-in-the-code)
9. [How to Run](#how-to-run)
10. [Key Takeaways](#key-takeaways)

---

## What Is a Race Condition?

A **race condition** occurs when multiple goroutines (or processes/threads) access and modify the **same shared resource concurrently**, and the final result depends on the unpredictable timing/order in which they run.

As the code comment explains:

> When multiple processes use the same or single resources, modification of that resource is not atomic. There's a high chance that if Process 1 changes the resource, Process 2 also changes it at the same time, creating a conflict in the final result.

**Concretely, in this code:** the line `p.views += 1` is *not* a single atomic CPU operation — it actually involves three steps:
1. Read the current value of `p.views`.
2. Add `1` to it.
3. Write the new value back to `p.views`.

If two goroutines interleave these steps (e.g., both read `views = 5` before either writes back `6`), one increment gets **lost**. Multiply this across 100 concurrent goroutines, and the final count becomes unpredictable — sometimes 100, sometimes 99, sometimes much lower.

---

## What Is a Mutex?

A **Mutex** (short for "mutual exclusion") is a synchronization primitive that ensures only **one goroutine at a time** can access a particular section of code (a "critical section").

Go provides this via `sync.Mutex`, with two key methods:

| Method | Purpose |
|---|---|
| `mu.Lock()` | Acquires the lock. If another goroutine already holds it, this call **blocks** until it's released. |
| `mu.Unlock()` | Releases the lock, allowing another waiting goroutine to acquire it. |

Only the goroutine that called `Lock()` should call the matching `Unlock()`.

---

## The `post` Struct

```go
type post struct {
	views int
	mu    sync.Mutex
}
```

- `views` is the **shared resource** — a simple counter that multiple goroutines will increment concurrently.
- `mu` is a `sync.Mutex` embedded directly in the struct, dedicated to protecting `views`. This is the idiomatic Go pattern: **pair a mutex with the specific data it protects**, rather than using one global lock for unrelated data.

---

## The `inc` Method — Locking a Shared Resource

```go
func (p *post) inc(wg *sync.WaitGroup) {

	defer func() {
		p.mu.Unlock()
		wg.Done()
	}()

	p.mu.Lock()
	p.views += 1
}
```

**Step-by-step:**

1. `inc` is a method on `*post`, so it can modify the receiver's fields directly (`p.views`).
2. It accepts a pointer to a `sync.WaitGroup` so it can signal completion back to `main`.
3. `p.mu.Lock()` is called **immediately before** the modification — this is the critical section.
4. `p.views += 1` — the actual (non-atomic) increment, now safely protected: while one goroutine holds the lock, no other goroutine can enter this line for the same `post` instance.
5. The `defer` block (registered at the top of the function, but executed last) does two things when `inc` returns:
   - `p.mu.Unlock()` — releases the lock so the next waiting goroutine can proceed.
   - `wg.Done()` — tells the `WaitGroup` that this goroutine has finished.

---

## Why `Unlock()` Is Called Inside `defer`

The code includes an important design discussion as comments:

- You *could* write `p.mu.Unlock()` directly after `p.views += 1`, without `defer`. But if the function panics, returns early, or hits an error **before** reaching that line, `Unlock()` would never run — leaving the mutex **locked forever**. Every subsequent goroutine calling `p.mu.Lock()` would then block indefinitely, effectively deadlocking the whole program.
- By putting `p.mu.Unlock()` inside a `defer`, Go guarantees it **always runs when the function returns** — whether that's a normal return, an early return, or a panic. This makes the lock/unlock pair much safer.

> "Never lock the whole function or logic by MUTEX, it's a bad practice. Only lock that particular line that performs modification."

This is a key Go idiom: **keep critical sections as small as possible**. Locking more code than necessary increases contention and hurts performance, since other goroutines are forced to wait longer than needed.

The code also notes: **"Sometimes MUTEX creates a bottleneck situation"** — because a mutex serializes access, a resource that's locked very frequently (e.g., by hundreds of concurrent goroutines) can become a performance chokepoint, since goroutines pile up waiting their turn instead of running in parallel.

---

## The `main` Function — Spawning 100 Goroutines

```go
var wg sync.WaitGroup
myPost := post{views: 0}

for i := 0; i < 100; i++ {
	wg.Add(1)
	go myPost.inc(&wg)
}

wg.Wait()
fmt.Println(myPost.views)
```

**What's happening:**

1. `wg` is a `sync.WaitGroup`, used to wait for all 100 goroutines to finish before printing the result.
2. `myPost` is created with `views: 0`.
3. The loop runs 100 times:
   - `wg.Add(1)` increments the WaitGroup's internal counter by 1, registering one more goroutine to wait for.
   - `go myPost.inc(&wg)` launches `inc` concurrently as a goroutine.
4. `wg.Wait()` blocks `main` until all 100 goroutines have called `wg.Done()` (i.e., the internal counter reaches zero).
5. Finally, `myPost.views` is printed — and because of the mutex, this will **reliably be 100** every time, since each increment is now properly synchronized.

---

## What Happens Without a Mutex (Observed Behavior)

The comment block in the code shows real sample output from running this program **without proper locking**:

```
go run main.go
100
go run main.go
99
go run main.go
100
```

This inconsistency is the classic signature of a race condition: sometimes all 100 increments "land" correctly, but other times two or more goroutines overwrite each other's update, silently losing an increment. The bug is **non-deterministic** — it may not show up every run, which makes race conditions notoriously hard to catch through casual testing.

> Tip: Go's built-in race detector can catch this class of bug reliably. Run with:
> ```bash
> go run -race main.go
> ```
> This will report a warning if unsynchronized concurrent access to `views` is detected.

With the mutex in place (as in this code), the output becomes **deterministic**: `100`, every single time.

---

## Mutex Best Practices Highlighted in the Code

1. **Pair the mutex with the data it protects** (`views` and `mu` live together in the same struct).
2. **Lock only the critical section**, not the entire function — minimize how long the lock is held.
3. **Always release the lock**, even on error paths — use `defer mu.Unlock()` right after (or very close to) `mu.Lock()`.
4. **Be aware of the performance trade-off** — a mutex serializes access and can become a bottleneck under heavy contention; only use it where shared mutable state truly needs protecting.

---

## How to Run

```bash
go run main.go
```

Expected output (deterministic, thanks to the mutex):
```
100
```

To see what an *unprotected* version would look like, try removing `p.mu.Lock()` and the `Unlock()` call, then run the program multiple times — you should see varying, inconsistent output values less than or equal to 100.

---

## Key Takeaways

- A **race condition** happens when concurrent goroutines read-modify-write a shared value without synchronization, causing lost updates.
- `sync.Mutex` provides `Lock()`/`Unlock()` to ensure only one goroutine executes a critical section at a time.
- Use `defer mu.Unlock()` immediately after `mu.Lock()` so the lock is **always** released, even during early returns or panics.
- Keep the locked region as small as possible — lock only the specific line(s) that mutate shared state, not the whole function.
- `sync.WaitGroup` (`Add`, `Done`, `Wait`) is used here to make `main` wait for all spawned goroutines to finish before reading the final result.
- Mutexes fix correctness but introduce serialization — under very high contention they can become a performance bottleneck.
