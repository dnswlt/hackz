package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"modernc.org/sqlite"
)

// JSON is a generic column type that transparently marshals/unmarshals any
// value to/from a SQLite TEXT column. Implement driver.Valuer for writes and
// sql.Scanner for reads so callers never touch encoding/json at the call site.
type JSON[T any] struct{ V T }

func (j JSON[T]) Value() (driver.Value, error) {
	b, err := json.Marshal(j.V)
	return string(b), err
}

func (j *JSON[T]) Scan(src any) error {
	var b []byte
	switch s := src.(type) {
	case string:
		b = []byte(s)
	case []byte:
		b = s
	default:
		return fmt.Errorf("JSON.Scan: unsupported source type %T", src)
	}
	return json.Unmarshal(b, &j.V)
}

// sqliteConnector opens a new SQLite connection and applies pragmas before
// returning it to database/sql's pool. This ensures every connection —
// not just the first one — gets the desired settings.
var sqliteDrv = &sqlite.Driver{}

type sqliteConnector struct {
	dsn     string
	pragmas []string
}

func (c *sqliteConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := sqliteDrv.Open(c.dsn)
	if err != nil {
		return nil, err
	}
	ec, ok := conn.(driver.ExecerContext)
	if !ok {
		conn.Close()
		return nil, fmt.Errorf("sqlite driver connection does not implement ExecerContext")
	}
	for _, p := range c.pragmas {
		if _, err := ec.ExecContext(ctx, p, nil); err != nil {
			conn.Close()
			return nil, fmt.Errorf("exec %q: %w", p, err)
		}
	}
	return conn, nil
}

func (c *sqliteConnector) Driver() driver.Driver { return sqliteDrv }

func cmdHello(ctx context.Context) error {
	db, err := sql.Open("sqlite", "hello.db")
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS greetings (
		id      INTEGER PRIMARY KEY AUTOINCREMENT,
		message TEXT    NOT NULL,
		created TEXT    NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	messages := []string{
		"Hello, world!",
		"Bonjour, le monde!",
		"Hola, mundo!",
		"Hallo, Welt!",
	}
	now := time.Now().Format(time.RFC3339)
	for _, msg := range messages {
		if _, err := db.ExecContext(ctx, `INSERT INTO greetings (message, created) VALUES (?, ?)`, msg, now); err != nil {
			return fmt.Errorf("insert %q: %w", msg, err)
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT id, message, created FROM greetings ORDER BY id`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	fmt.Printf("%-4s  %-30s  %s\n", "ID", "Message", "Created")
	fmt.Printf("%-4s  %-30s  %s\n", "----", "------------------------------", "--------------------")
	for rows.Next() {
		var id int
		var message, created string
		if err := rows.Scan(&id, &message, &created); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		fmt.Printf("%-4d  %-30s  %s\n", id, message, created)
	}
	return rows.Err()
}

func cmdConcurrent(ctx context.Context) error {
	// Use a custom connector so every pooled connection gets the pragmas,
	// not just whichever one happens to run the first Exec.
	db := sql.OpenDB(&sqliteConnector{
		dsn: "concurrent.db",
		pragmas: []string{
			`PRAGMA journal_mode=WAL`,
			`PRAGMA synchronous=NORMAL`,
			`PRAGMA busy_timeout=5000`,
		},
	})
	defer db.Close()

	var err error

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS events (
		id        INTEGER PRIMARY KEY AUTOINCREMENT,
		worker_id INTEGER NOT NULL,
		seq       INTEGER NOT NULL,
		ts        TEXT    NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	const (
		numWorkers = 10
		numInserts = 20
	)

	var wg sync.WaitGroup
	errc := make(chan error, numWorkers)

	for w := range numWorkers {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for seq := range numInserts {
				_, err := db.ExecContext(ctx,
					`INSERT INTO events (worker_id, seq, ts) VALUES (?, ?, ?)`,
					workerID, seq, time.Now().Format(time.RFC3339Nano),
				)
				if err != nil {
					errc <- fmt.Errorf("worker %d seq %d: %w", workerID, seq, err)
					return
				}
			}
		}(w)
	}

	wg.Wait()
	close(errc)
	if err := <-errc; err != nil {
		return err
	}

	rows, err := db.QueryContext(ctx,
		`SELECT worker_id, COUNT(*) AS n, MIN(ts), MAX(ts)
		 FROM events
		 GROUP BY worker_id
		 ORDER BY worker_id`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	fmt.Printf("%-9s  %-6s  %-35s  %s\n", "Worker", "Rows", "First write", "Last write")
	fmt.Printf("%-9s  %-6s  %-35s  %s\n", "---------", "------", "-----------------------------------", "-----------------------------------")
	for rows.Next() {
		var workerID, n int
		var minTS, maxTS string
		if err := rows.Scan(&workerID, &n, &minTS, &maxTS); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		fmt.Printf("%-9d  %-6d  %-35s  %s\n", workerID, n, minTS, maxTS)
	}
	return rows.Err()
}

type Attributes struct {
	Color    string   `json:"color"`
	WeightKg float64  `json:"weight_kg"`
	Tags     []string `json:"tags"`
}

func cmdJSON(ctx context.Context) error {
	db, err := sql.Open("sqlite", "json.db")
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS products (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		name       TEXT NOT NULL,
		attributes TEXT NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}

	products := []struct {
		name string
		attr Attributes
	}{
		{"Bicycle", Attributes{Color: "red", WeightKg: 8.5, Tags: []string{"sport", "outdoor"}}},
		{"Laptop", Attributes{Color: "silver", WeightKg: 1.4, Tags: []string{"tech", "work"}}},
		{"Tent", Attributes{Color: "green", WeightKg: 2.1, Tags: []string{"outdoor", "camping"}}},
	}
	for _, p := range products {
		_, err := db.ExecContext(ctx,
			`INSERT INTO products (name, attributes) VALUES (?, ?)`,
			p.name, JSON[Attributes]{p.attr},
		)
		if err != nil {
			return fmt.Errorf("insert %q: %w", p.name, err)
		}
	}

	rows, err := db.QueryContext(ctx, `SELECT id, name, attributes FROM products ORDER BY id`)
	if err != nil {
		return fmt.Errorf("query: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		var attr JSON[Attributes]
		if err := rows.Scan(&id, &name, &attr); err != nil {
			return fmt.Errorf("scan: %w", err)
		}
		fmt.Printf("[%d] %s — color=%s weight=%.1fkg tags=%v\n",
			id, name, attr.V.Color, attr.V.WeightKg, attr.V.Tags)
	}
	return rows.Err()
}

func cmdTxn(ctx context.Context) error {
	// No busy_timeout: we want conflicts to surface immediately, not be retried.
	db := sql.OpenDB(&sqliteConnector{
		dsn:     "txn.db",
		pragmas: []string{`PRAGMA journal_mode=WAL`},
	})
	defer db.Close()

	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS accounts (
		id      INTEGER PRIMARY KEY,
		name    TEXT    NOT NULL,
		balance INTEGER NOT NULL
	)`)
	if err != nil {
		return fmt.Errorf("create table: %w", err)
	}
	_, err = db.ExecContext(ctx, `INSERT OR REPLACE INTO accounts VALUES (1, 'alice', 0)`)
	if err != nil {
		return fmt.Errorf("seed: %w", err)
	}

	// Two connections = two independent transaction snapshots.
	conn1, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn1.Close()
	conn2, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer conn2.Close()

	// Open both transactions before either writes, so they share the same
	// snapshot of the DB. *sql.Tx is not goroutine-bound; we drive both from
	// here sequentially.
	tx1, err := conn1.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("tx1 begin: %w", err)
	}
	tx2, err := conn2.BeginTx(ctx, nil)
	if err != nil {
		tx1.Rollback()
		return fmt.Errorf("tx2 begin: %w", err)
	}

	var bal1, bal2 int
	if err := tx1.QueryRowContext(ctx, `SELECT balance FROM accounts WHERE id=1`).Scan(&bal1); err != nil {
		tx1.Rollback(); tx2.Rollback()
		return fmt.Errorf("tx1 read: %w", err)
	}
	if err := tx2.QueryRowContext(ctx, `SELECT balance FROM accounts WHERE id=1`).Scan(&bal2); err != nil {
		tx1.Rollback(); tx2.Rollback()
		return fmt.Errorf("tx2 read: %w", err)
	}
	fmt.Printf("tx1 read balance=%d\n", bal1)
	fmt.Printf("tx2 read balance=%d\n", bal2)

	// tx1 writes and commits first.
	if _, err := tx1.ExecContext(ctx, `UPDATE accounts SET balance=? WHERE id=1`, bal1+100); err != nil {
		tx1.Rollback(); tx2.Rollback()
		return fmt.Errorf("tx1 write: %w", err)
	}
	if err := tx1.Commit(); err != nil {
		tx1.Rollback(); tx2.Rollback()
		return fmt.Errorf("tx1 commit: %w", err)
	}
	fmt.Println("tx1 committed balance+100")

	// tx2 now tries to write against its stale snapshot.
	if _, err := tx2.ExecContext(ctx, `UPDATE accounts SET balance=? WHERE id=1`, bal2+200); err != nil {
		tx2.Rollback()
		fmt.Printf("tx2 write failed (expected): %v\n", err)
	} else if err := tx2.Commit(); err != nil {
		tx2.Rollback()
		fmt.Printf("tx2 commit failed (expected): %v\n", err)
	} else {
		fmt.Println("tx2 committed (no conflict detected — lost update!)")
	}

	var final int
	if err := db.QueryRowContext(ctx, `SELECT balance FROM accounts WHERE id=1`).Scan(&final); err != nil {
		return err
	}
	fmt.Printf("final balance=%d (expected 100 if G1 won)\n", final)
	return nil
}

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "usage: gosql <command>\n\nCommands:\n  hello       create hello.db, insert greetings, and print them\n  concurrent  10 goroutines write concurrently to concurrent.db\n  json        store and retrieve structured objects as JSON columns\n  txn         two goroutines write the same row; observe the conflict\n")
		os.Exit(1)
	}

	ctx := context.Background()
	var err error
	switch os.Args[1] {
	case "hello":
		err = cmdHello(ctx)
	case "concurrent":
		err = cmdConcurrent(ctx)
	case "json":
		err = cmdJSON(ctx)
	case "txn":
		err = cmdTxn(ctx)
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %q\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		log.Fatalf("error: %v", err)
	}
}
