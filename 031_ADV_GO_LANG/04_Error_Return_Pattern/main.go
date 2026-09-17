package main

import (
	"fmt"
	"strconv"
)

func main() {

	// GO don't use exceptions for normal failures
	// In Go Functions return errors as normal values.

	// Value, err := something ()
	// if err != nil {handle the error}

}

func run() error {

}

func parseLevel(s string) (int, error) {
	// (value, error)
	// if nil error -> success
	// if not nil -> failure

	// pattern
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("Level Must be a number")
	}

}
