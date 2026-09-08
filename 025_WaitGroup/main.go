package main

import (
	"fmt"
	"sync"
)

func task(id int, w *sync.WaitGroup) {
	defer w.Done()
	fmt.Println("Doing task", id)
}

func main() {

	// Here we are creating the WaitGroup variable, which will be used to wait for all the goroutines to finish.
	var wg sync.WaitGroup

	for i := 0; i <= 10; i++ {
		wg.Add(1)
		go task(i, &wg)
	}
	wg.Wait()

}
