package main

import (
	"flag"
	"fmt"
	"reflect"
	"sync"
	"sync/atomic"
	"time"
)

type Counter interface {
	Incr()
	Value() int64
}

type MutexCounter struct {
	mut sync.Mutex
	n   int64
}

func (c *MutexCounter) Incr() {
	c.mut.Lock()
	c.n++
	c.mut.Unlock()
}

func (c *MutexCounter) Value() int64 {
	c.mut.Lock()
	result := c.n
	c.mut.Unlock()
	return result
}

type AtomicCounter struct {
	n atomic.Int64
}

func (c *AtomicCounter) Incr() {
	c.n.Add(1)
}

func (c *AtomicCounter) Value() int64 {
	return c.n.Load()
}

type ShardedAtomicCounter struct {
	n []atomic.Int64
}

func NewShardedAtomicCounter(shards int) *ShardedAtomicCounter {
	return &ShardedAtomicCounter{
		n: make([]atomic.Int64, shards),
	}
}

func (c *ShardedAtomicCounter) Incr(shard int) {
	c.n[shard].Add(1)
}

func (c *ShardedAtomicCounter) Value() int64 {
	var v int64
	for i := range c.n {
		v += c.n[i].Load()
	}
	return v
}

func (c *ShardedAtomicCounter) Shards() int {
	return len(c.n)
}

func runConcurrentTest(f func(goroNum int), goroCount int) time.Duration {
	var done sync.WaitGroup
	var ready sync.WaitGroup
	barrier := make(chan struct{})
	for i := range goroCount {
		done.Add(1)
		ready.Add(1)
		go func() {
			defer done.Done()
			ready.Done()
			<-barrier
			f(i)
		}()
	}

	// Wait until all goros are ready.
	ready.Wait()
	// Open the flood gates, i.e. close the barrier o_O.
	started := time.Now()
	close(barrier)
	// Wait until all goros are done.
	done.Wait()

	return time.Since(started)
}

func runShardedTest(counter *ShardedAtomicCounter, goroCount, iterCount int) {
	n := counter.Shards()

	d := runConcurrentTest(func(goroNum int) {
		for i := range iterCount {
			counter.Incr((goroNum*iterCount + i) % n)
		}
	}, goroCount)

	ok := counter.Value() == int64(goroCount*iterCount)
	t := reflect.TypeOf(counter)
	fmt.Printf("Counter type %v took %.3f seconds. Counter value: %d (ok=%t)\n",
		t, d.Seconds(), counter.Value(), ok)
}

func runTest(counter Counter, goroCount, iterCount int) {
	d := runConcurrentTest(func(goroNum int) {
		for range iterCount {
			counter.Incr()
		}
	}, goroCount)

	ok := counter.Value() == int64(goroCount*iterCount)
	t := reflect.TypeOf(counter)
	fmt.Printf("Counter type %v took %.3f seconds. Counter value: %d (ok=%t)\n",
		t, d.Seconds(), counter.Value(), ok)
}

func main() {

	goroCount := flag.Int("goroutines", 500, "Number of concurrent goroutines to run")
	iterCount := flag.Int("iterations", 100_000, "Number of iterations each goroutine executes")
	shardCount := flag.Int("shards", 0, "Number of shards to use in sharded atomic counter. (0 uses as many shards as there are goroutines.)")
	flag.Parse()

	if *shardCount == 0 {
		*shardCount = *goroCount
	}

	var counters []Counter = []Counter{
		&MutexCounter{},
		&AtomicCounter{},
	}

	fmt.Printf("Using %d goroutines, %d iterations, %d shards\n", *goroCount, *iterCount, *shardCount)
	for _, counter := range counters {
		runTest(counter, *goroCount, *iterCount)
	}
	runShardedTest(NewShardedAtomicCounter(*shardCount), *goroCount, *iterCount)
}
