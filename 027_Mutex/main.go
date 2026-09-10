package main

import (
	"fmt"
	"sync"
)

// When we do multiThreading then to avoide the raise condition we use MUTEX

/* What is raise condition
When multiple processes use the same or single resources then modification on that resources is bot atomic.
There is a hight chancet that if Process 1 chance the resource then process 2 also change the resource so it create conflict in the final result.
*/

type post struct {
	views int
	// Here we add mutex in views resource.
	mu sync.Mutex
}

func (p *post) inc(wg *sync.WaitGroup) {
	defer wg.Done()
	p.views += 1
}

func main() {

	var wg sync.WaitGroup

	myPost := post{views: 0}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go myPost.inc(&wg)
	}

	wg.Wait()

	fmt.Println(myPost.views)

	/*
		If we see the output we notice that sometime 100 then 99 then 67 the 89 so that views resource is use my many processors and they make changes in them that why we get our outpi]ut like this.

		Due to goRoutines and concurrency the views filed value modify multiple at sametime.
		So we get like this value.
		This condition is konwn as RAISE CONDITION.

		PS C:\Users\ALOK\Desktop\GO\GOLANG\027_Mutex> go run main.go
		100
		PS C:\Users\ALOK\Desktop\GO\GOLANG\027_Mutex> go run main.go
		99
		PS C:\Users\ALOK\Desktop\GO\GOLANG\027_Mutex>
		PS C:\Users\ALOK\Desktop\GO\GOLANG\027_Mutex> go run main.go
		100
	*/

	// To Resolve this problem we use Mutex when we use goRoutines and want many processors to process a single resources multiple times.
	// In MUTEX when that particular resource is in USE then we LOCK that resource so that no other processor use that resource and manuplate the values.

}
