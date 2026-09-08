package main

import (
	"fmt"
	"time"
)

func task(id int) {
	fmt.Println("Doing task", id)
}

func brewCoffee() {
	fmt.Println("Brewing coffee...")
}

func toastBread() {
	fmt.Println("Toasting bread...")
}

func main() {
	for i := 0; i <= 10; i++ {
		go task(i)
	}

	go brewCoffee()
	go toastBread()
	fmt.Println("Breakfast ready!")

	time.Sleep(time.Second * 2)

}
