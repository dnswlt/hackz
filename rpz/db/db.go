package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type DB struct {
	pool *pgxpool.Pool
}

type Item struct {
	ID         string
	Name       string
	Payload    string
	CreateTime time.Time
}

func New(ctx context.Context, connStr string) (*DB, error) {
	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		return nil, err
	}
	return &DB{pool: pool}, nil
}

func (db *DB) Close() {
	db.pool.Close()
}

func (db *DB) BatchInsertItems(ctx context.Context, items []Item) error {
	if len(items) == 0 {
		return nil
	}

	rows := make([][]any, len(items))
	for i, item := range items {
		rows[i] = []any{item.ID, item.Name, item.Payload, item.CreateTime}
	}

	_, err := db.pool.CopyFrom(
		ctx,
		pgx.Identifier{"items"},
		[]string{"id", "name", "payload", "create_time"},
		pgx.CopyFromRows(rows),
	)
	return err
}

func (db *DB) InsertItem(ctx context.Context, item Item) error {
	if item.CreateTime.IsZero() {
		item.CreateTime = time.Now().UTC()
	}
	const query = `
			INSERT INTO items (id, name, payload, create_time)
			VALUES ($1, $2, $3, $4)
		`

	_, err := db.pool.Exec(ctx, query, item.ID, item.Name, item.Payload, item.CreateTime)
	return err
}

func (db *DB) GetItemByID(ctx context.Context, id string) (*Item, error) {
	const query = `
		SELECT id, name, payload, create_time
		FROM items
		WHERE id = $1
	`

	row := db.pool.QueryRow(ctx, query, id)

	var item Item
	err := row.Scan(&item.ID, &item.Name, &item.Payload, &item.CreateTime)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // Not found
		}
		return nil, err
	}
	return &item, nil
}

func (db *DB) GetAllItemIDs(ctx context.Context) ([]string, error) {
	const query = `SELECT id FROM items`

	rows, err := db.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}
