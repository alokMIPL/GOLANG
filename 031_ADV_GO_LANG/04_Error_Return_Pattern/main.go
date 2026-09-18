package main

import (
	"fmt"
	"log"
	"strconv"
)

func main() {

	// GO don't use exceptions for normal failures
	// In Go Functions return errors as normal values.

	// Value, err := something ()
	// if err != nil {handle the error}

	if err := run(); err != nil {
		log.Fatal(err)
	}

}

func run() error {

	input := "3"
	level, err := parseLevel(input)
	if err != nil {
		return err
	}

	fmt.Println("Selected level", level)
	return nil

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

	if n < 1 || n > 5 {
		return 0, fmt.Errorf("Level must be 1 and 5")
	}

	return n, nil

}
