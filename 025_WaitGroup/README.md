What a WaitGroup Actually Is

A sync.WaitGroup is essentially a counter with three operations:

Method	What it does
wg.Add(n)	Increases the counter by n
wg.Done()	Decreases the counter by 1 (shorthand for wg.Add(-1))
wg.Wait()	Blocks the calling goroutine until the counter reaches 0


WaitGroup must be passed by pointer (*sync.WaitGroup), not by value. If you accidentally did:



Summary
Step	What happens
wg.Add(1) × 11	Counter goes from 0 → 11, once per goroutine launched
11 goroutines run concurrently	Each does its work, then calls Done() when finished
wg.Done() × 11	Counter decreases by 1 each time a goroutine finishes
wg.Wait()	Blocks main() until counter hits 0 — i.e., all goroutines have completed



Step 1: Think of WaitGroup as a Counter

Imagine a simple number that starts at 0. That's all a WaitGroup really is internally — a counter.

counter = 0
Step 2: Add(1) Increases the Counter

Every time you call wg.Add(1), the counter goes up by 1.

In your code, this happens inside a loop that runs 11 times:

go
for i := 0; i <= 10; i++ {
	wg.Add(1)        // counter: 0→1→2→3...→11
	go task(i, &wg)  // start a goroutine
}

So after the loop finishes, counter = 11, and there are 11 goroutines now running in the background.

Step 3: Done() Decreases the Counter

Each task() goroutine, when it finishes its work, calls w.Done():

go
func task(id int, w *sync.WaitGroup) {
	defer w.Done()   // this runs when task() finishes — decreases counter by 1
	fmt.Println("Doing task", id)
}

So as each of the 11 goroutines finishes (in random order, since they run concurrently), the counter drops:

counter: 11 → 10 → 9 → 8 → ... → 1 → 0
Step 4: wg.Wait() — This Is the Part to Understand Carefully

Right after the loop, your code calls:

go
wg.Wait()

Important: this line runs immediately — right after the loop ends, not after all goroutines are done. At the moment it's called, the counter could still be 11, or 9, or anything — some goroutines may not have even started yet.

What wg.Wait() does is:

"Pause main() right here, and don't move to the next line, until the counter becomes 0."

So main() literally freezes at that line. It's not doing anything else, not checking again and again — it's just paused, waiting.

Then, in the background, goroutines keep finishing and calling Done(), dropping the counter lower and lower.

The moment the counter hits exactly 0 (meaning all 11 goroutines have finished), wg.Wait() unblocks, and main() is allowed to continue to whatever comes after it.