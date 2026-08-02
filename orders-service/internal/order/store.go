package order

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

var ErrNotFound = errors.New("order not found")

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	const ddl = `
CREATE TABLE IF NOT EXISTS orders (
  id TEXT PRIMARY KEY,
  items_json TEXT NOT NULL,
  total INTEGER NOT NULL,
  status TEXT NOT NULL,
  timestamp TEXT NOT NULL
);`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

func (s *Store) Create(ctx context.Context, o Order) error {
	itemsJSON, err := json.Marshal(o.Items)
	if err != nil {
		return fmt.Errorf("marshal items: %w", err)
	}

	_, err = s.db.ExecContext(
		ctx,
		`INSERT INTO orders (id, items_json, total, status, timestamp) VALUES (?, ?, ?, ?, ?)`,
		o.ID,
		string(itemsJSON),
		o.Total,
		o.Status,
		o.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

func (s *Store) GetByID(ctx context.Context, id string) (Order, error) {
	var (
		o         Order
		itemsJSON string
		ts        string
	)

	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, items_json, total, status, timestamp FROM orders WHERE id = ?`,
		id,
	).Scan(&o.ID, &itemsJSON, &o.Total, &o.Status, &ts)
	if errors.Is(err, sql.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("get order: %w", err)
	}

	if err := json.Unmarshal([]byte(itemsJSON), &o.Items); err != nil {
		return Order{}, fmt.Errorf("unmarshal items: %w", err)
	}

	o.Timestamp, err = time.Parse(time.RFC3339Nano, ts)
	if err != nil {
		return Order{}, fmt.Errorf("parse timestamp: %w", err)
	}
	return o, nil
}
