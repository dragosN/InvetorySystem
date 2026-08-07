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

// OutboxRow is an unpublished domain event waiting for Kafka.
type OutboxRow struct {
	ID        int64
	EventID   string
	OrderID   string
	Payload   []byte
	CreatedAt time.Time
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// SQLite writers serialize; one open connection avoids "database is locked" races.
	db.SetMaxOpenConns(1)

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma wal: %w", err)
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pragma busy_timeout: %w", err)
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
);

CREATE TABLE IF NOT EXISTS outbox (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  event_id TEXT NOT NULL UNIQUE,
  order_id TEXT NOT NULL,
  payload TEXT NOT NULL,
  created_at TEXT NOT NULL,
  published_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_outbox_unpublished
  ON outbox(id)
  WHERE published_at IS NULL;
`
	if _, err := s.db.Exec(ddl); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	return nil
}

// CreateWithOutbox inserts the order and its outbox event in one transaction.
// The HTTP handler returns after this commit; Kafka publish is async.
func (s *Store) CreateWithOutbox(ctx context.Context, o Order, eventID string, payload []byte) error {
	itemsJSON, err := json.Marshal(o.Items)
	if err != nil {
		return fmt.Errorf("marshal items: %w", err)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(
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

	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO outbox (event_id, order_id, payload, created_at) VALUES (?, ?, ?, ?)`,
		eventID,
		o.ID,
		string(payload),
		o.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insert outbox: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
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

func (s *Store) ListUnpublishedOutbox(ctx context.Context, limit int) ([]OutboxRow, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.QueryContext(
		ctx,
		`SELECT id, event_id, order_id, payload, created_at
		 FROM outbox
		 WHERE published_at IS NULL
		 ORDER BY id ASC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list outbox: %w", err)
	}
	defer rows.Close()

	var out []OutboxRow
	for rows.Next() {
		var (
			r  OutboxRow
			ts string
		)
		if err := rows.Scan(&r.ID, &r.EventID, &r.OrderID, &r.Payload, &ts); err != nil {
			return nil, fmt.Errorf("scan outbox: %w", err)
		}
		r.CreatedAt, err = time.Parse(time.RFC3339Nano, ts)
		if err != nil {
			return nil, fmt.Errorf("parse outbox created_at: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) MarkOutboxPublished(ctx context.Context, id int64, at time.Time) error {
	return s.MarkOutboxPublishedBatch(ctx, []int64{id}, at)
}

func (s *Store) MarkOutboxPublishedBatch(ctx context.Context, ids []int64, at time.Time) error {
	if len(ids) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin mark outbox: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	ts := at.UTC().Format(time.RFC3339Nano)
	stmt, err := tx.PrepareContext(ctx, `UPDATE outbox SET published_at = ? WHERE id = ? AND published_at IS NULL`)
	if err != nil {
		return fmt.Errorf("prepare mark outbox: %w", err)
	}
	defer stmt.Close()

	for _, id := range ids {
		if _, err := stmt.ExecContext(ctx, ts, id); err != nil {
			return fmt.Errorf("mark outbox published id=%d: %w", id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit mark outbox: %w", err)
	}
	return nil
}

func (s *Store) CountUnpublishedOutbox(ctx context.Context) (int64, error) {
	var n int64
	err := s.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM outbox WHERE published_at IS NULL`,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count outbox: %w", err)
	}
	return n, nil
}
