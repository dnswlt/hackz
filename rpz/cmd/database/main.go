package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/dnswlt/hackz/rpz"
	"github.com/dnswlt/hackz/rpz/db"
)

func insertItems(ctx context.Context, client *db.DB, itemCount, payloadLen int) ([]string, error) {
	const batchSize = 1024
	batch := make([]db.Item, batchSize)

	now := time.Now().UTC()

	insertedItemIDs := make([]string, 0, itemCount)

	for len(insertedItemIDs) < itemCount {
		b := min(itemCount-len(insertedItemIDs), batchSize)
		k := len(insertedItemIDs)
		for i := range b {
			batch[i] = db.Item{
				ID:         fmt.Sprintf("item%06d", k+i),
				Name:       fmt.Sprintf("Name %d", k+i),
				Payload:    rpz.RandomString(payloadLen),
				CreateTime: now,
			}
			insertedItemIDs = append(insertedItemIDs, batch[i].ID)
		}
		err := client.BatchInsertItems(ctx, batch[:b])
		if err != nil {
			return nil, fmt.Errorf("failed to batch insert items: %v", err)
		}
	}
	return insertedItemIDs, nil
}

func queryRandomItems(ctx context.Context, client *db.DB, itemIDs []string, n int) error {
	for range n {
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		j := rand.Intn(len(itemIDs))
		item, err := client.GetItemByID(ctx, itemIDs[j])
		cancel()
		if err != nil {
			return fmt.Errorf("error querying item %q: %v", itemIDs[j], err)
		}
		if item == nil {
			return fmt.Errorf("item with ID %q not found", itemIDs[j])
		}
	}
	return nil
}

func main() {

	payloadLen := flag.Int("payload-len", 1024, "Length of the payload string.")
	itemCount := flag.Int("items", 1024, "Number of items to insert before querying. Set to 0 to insert nothing.")
	queryCount := flag.Int("queries", 4096, "Number of queries to run per goroutine during benchmark.")
	goroCount := flag.Int("goroutines", 1, "Number of goroutines concurrently querying the database.")
	conn := flag.String("conn", "postgres://loadtest_client@localhost/loadtest",
		"PostgreSQL connection string. Set to '' to use PGUSER etc. env vars instead.")

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	client, err := db.New(ctx, *conn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	var itemIDs []string
	if *itemCount > 0 {
		var err error
		// Batch insert
		insertStart := time.Now()
		itemIDs, err = insertItems(ctx, client, *itemCount, *payloadLen)
		if err != nil {
			log.Fatalf("Error during batch insert: %v", err)
		}
		log.Printf("Took %.3f to insert %d items", time.Since(insertStart).Seconds(), len(itemIDs))
	} else {
		// Retrieve existing item IDs to query.
		var err error
		itemIDs, err = client.GetAllItemIDs(ctx)
		if err != nil {
			log.Fatalf("Error reading item IDs: %v", err)
		}
		log.Printf("Read %d item IDs from the database.", len(itemIDs))
	}

	// Query by ID
	queryStart := time.Now()

	var wg sync.WaitGroup
	wg.Add(*goroCount)

	errCh := make(chan error)
	queryCtx, queryCancel := context.WithCancel(context.Background())
	for range *goroCount {
		go func() {
			defer wg.Done()
			err := queryRandomItems(queryCtx, client, itemIDs, *queryCount)
			if err != nil {
				select {
				case errCh <- err:
				case <-queryCtx.Done():
				}
			}
		}()
	}

	coordCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(coordCh)
	}()

	select {
	case err := <-errCh:
		queryCancel() // Cancel all others
		<-coordCh
		log.Printf("Error in one of the goroutines: %v", err)
	case <-coordCh:
		n := *queryCount * *goroCount
		log.Printf("Executed %d queries in %v", n, time.Since(queryStart))
	}
}
