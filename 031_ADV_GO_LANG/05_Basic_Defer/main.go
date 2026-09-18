package main

import (
	"errors"
	"fmt"
)

func main() {

	// defer resp.body.close()

	fmt.Println("Case 1: success")
	if err := doWork(true); err != nil {
		fmt.Println("error:", err)
	}

	fmt.Println("Case 2: Fail Early")
	if err := doWork(false); err != nil {
		fmt.Println("error:", err)
	}

}

func doWork(success bool) error {
	// resource related
	// start message -> resource acquired
	// cleanup message -> resource released

	fmt.Println("start: resource acquired")

	// this defer will guranntee this runs at the end of the func
	// it return both the path at success as well as for errors.
	defer fmt.Println("cleanup: resource released")

	if !success {
		return errors.New("Something went wrong. i am returning early.")
	}

	fmt.Println("work: doing something imp")
	fmt.Println("work: this work is done")

	return nil

}
