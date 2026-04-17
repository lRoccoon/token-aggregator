package main

import (
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

type Record struct {
	DeviceID    string
	Source      string
	Date        string
	TotalTokens int64
	CostUSD     float64
}

type Store struct {
	db *sql.DB
}

const schemaSQL = `
CREATE TABLE IF NOT EXISTS usage (
  device_id   TEXT NOT NULL,
  source      TEXT NOT NULL,
  date        TEXT NOT NULL,
  total_tokens INTEGER NOT NULL DEFAULT 0,
  cost_usd    REAL NOT NULL DEFAULT 0,
  updated_at  INTEGER NOT NULL,
  PRIMARY KEY (device_id, source, date)
);
CREATE INDEX IF NOT EXISTS idx_usage_date ON usage(date);
`

func OpenStore(path string) (*Store, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schemaSQL); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) Upsert(records []Record) error {
	if len(records) == 0 {
		return nil
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	stmt, err := tx.Prepare(`
		INSERT INTO usage(device_id, source, date, total_tokens, cost_usd, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(device_id, source, date) DO UPDATE SET
			total_tokens = excluded.total_tokens,
			cost_usd     = excluded.cost_usd,
			updated_at   = excluded.updated_at`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	now := time.Now().Unix()
	for _, r := range records {
		if _, err := stmt.Exec(r.DeviceID, r.Source, r.Date, r.TotalTokens, r.CostUSD, now); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type SourceAgg struct {
	TodayTokens int64   `json:"today_tokens"`
	TotalTokens int64   `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost_usd"`
}

type DeviceAgg struct {
	TodayTokens int64                `json:"today_tokens"`
	TotalTokens int64                `json:"total_tokens"`
	TotalCost   float64              `json:"total_cost_usd"`
	Sources     map[string]SourceAgg `json:"sources"`
}

type Report struct {
	Today       string               `json:"today"`
	UpdatedAt   int64                `json:"updated_at"`
	TodayTokens int64                `json:"today_tokens"`
	TotalTokens int64                `json:"total_tokens"`
	TotalCost   float64              `json:"total_cost_usd"`
	Devices     map[string]DeviceAgg `json:"devices"`
}

func (s *Store) BuildReport(today string) (*Report, error) {
	rows, err := s.db.Query(`SELECT device_id, source, date, total_tokens, cost_usd FROM usage`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	r := &Report{
		Today:     today,
		UpdatedAt: time.Now().Unix(),
		Devices:   map[string]DeviceAgg{},
	}
	for rows.Next() {
		var rec Record
		if err := rows.Scan(&rec.DeviceID, &rec.Source, &rec.Date, &rec.TotalTokens, &rec.CostUSD); err != nil {
			return nil, err
		}
		dev, ok := r.Devices[rec.DeviceID]
		if !ok {
			dev = DeviceAgg{Sources: map[string]SourceAgg{}}
		}
		sa := dev.Sources[rec.Source]
		sa.TotalTokens += rec.TotalTokens
		sa.TotalCost += rec.CostUSD
		if rec.Date == today {
			sa.TodayTokens += rec.TotalTokens
			dev.TodayTokens += rec.TotalTokens
			r.TodayTokens += rec.TotalTokens
		}
		dev.TotalTokens += rec.TotalTokens
		dev.TotalCost += rec.CostUSD
		dev.Sources[rec.Source] = sa
		r.Devices[rec.DeviceID] = dev
		r.TotalTokens += rec.TotalTokens
		r.TotalCost += rec.CostUSD
	}
	return r, rows.Err()
}
