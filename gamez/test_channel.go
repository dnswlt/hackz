package main

import (
	"fmt"
)

func main() {
	// range over channel
	ch1 := make(chan int)
	go func() {
		for i := 0; i < 5; i++ {
			ch1 <- i
		}
		// If you don't close the channel, the main goroutine ranging over
		// the channel will hang forever.
		close(ch1)
	}()
	for v := range ch1 {
		fmt.Printf("%d\n", v)
	}
	fmt.Printf("---Done ranging.\n")

	// multiple return values to determine if channel was closed.
	ch2 := make(chan int)
	go func() {
		ch2 <- 1
		ch2 <- 2
		// Must close ch2 here, else main goroutine hangs.
		close(ch2)
	}()
	ok := true
	for ok {
		var v int
		v, ok = <-ch2
		fmt.Printf("%d, %t\n", v, ok)
	}
	fmt.Printf("---Done with two-value assignment.\n")

	// select on closed channel yields the zero value!
	ch3 := make(chan int)
	go func() {
		close(ch3)
	}()
	select {
	case x := <-ch3:
		fmt.Printf("%d\n", x)
	}
	fmt.Printf("---Done with select.\n")

	ch4 := make(chan int)
	close(ch4)
	v := <-ch4
	fmt.Printf("%d\n", v)
	fmt.Printf("---Done reading from closed channel.\n")
}
