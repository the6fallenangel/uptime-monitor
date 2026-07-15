package storage

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/the6fallenangel/uptime-monitor/internal/models"
)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(ctx context.Context, connString string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(ctx, connString)
	if err != nil {
		return nil, fmt.Errorf("creating connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("pinging database: %w", err)
	}

	if err := migrate(ctx, pool); err != nil {
		return nil, fmt.Errorf("running migrations: %w", err)
	}

	return &PostgresStorage{pool: pool}, nil
}

func (s *PostgresStorage) Close() {
	s.pool.Close()
}

func migrate(ctx context.Context, pool *pgxpool.Pool) error {
	const schema = `
	CREATE TABLE IF NOT EXISTS monitors (
		id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		name TEXT NOT NULL,
		url TEXT NOT NULL,
		interval_seconds INTEGER NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);
	
	CREATE TABLE IF NOT EXISTS checks (
		id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		monitor_id BIGINT NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
		status TEXT NOT NULL,
		status_code INTEGER,
		response_time_ms INTEGER NOT NULL,
		error TEXT NOT NULL DEFAULT '',
		checked_at TIMESTAMPTZ NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_checks_monitor_id ON checks(monitor_id);
	CREATE INDEX IF NOT EXISTS idx_checks_monitor_time ON checks(monitor_id, checked_at DESC);`

	_, err := pool.Exec(ctx, schema)
	return err
}

func (s *PostgresStorage) CreateMonitor(ctx context.Context, m models.Monitor) (models.Monitor, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO monitors (name, url, interval_seconds)
		 VALUES ($1, $2, $3)
		 RETURNING id, created_at`,
		m.Name, m.URL, int(m.Interval.Seconds()),
	)
	if err := row.Scan(&m.ID, &m.CreatedAt); err != nil {
		return models.Monitor{}, fmt.Errorf("inserting monitor: %w", err)
	}
	return m, nil
}

func (s *PostgresStorage) ListMonitors(ctx context.Context) ([]models.Monitor, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, url, interval_seconds, created_at FROM monitors ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("querying monitors: %w", err)
	}
	defer rows.Close()

	var monitors []models.Monitor
	for rows.Next() {
		var m models.Monitor
		var intervalSeconds int

		if err := rows.Scan(&m.ID, &m.Name, &m.URL, &intervalSeconds, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scanning monitor row: %w", err)
		}
		m.Interval = time.Duration(intervalSeconds) * time.Second
		monitors = append(monitors, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating monitor rows: %w", err)
	}
	return monitors, nil
}

func (s *PostgresStorage) GetMonitor(ctx context.Context, id int64) (models.Monitor, error) {
	var m models.Monitor
	var intervalSeconds int

	row := s.pool.QueryRow(ctx,
		`SELECT id, name, url, interval_seconds, created_at FROM monitors WHERE id = $1`, id)

	if err := row.Scan(&m.ID, &m.Name, &m.URL, &intervalSeconds, &m.CreatedAt); err != nil {
		return models.Monitor{}, fmt.Errorf("monitor with id %d not found: %w", id, err)
	}
	m.Interval = time.Duration(intervalSeconds) * time.Second
	return m, nil
}

func (s *PostgresStorage) DeleteMonitor(ctx context.Context, id int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM monitors WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deleting monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("monitor with id %d not found", id)
	}
	return nil
}

func (s *PostgresStorage) SaveCheck(ctx context.Context, c models.Check) (models.Check, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO checks (monitor_id, status, status_code, response_time_ms, error, checked_at)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING id`,
		c.MonitorID, c.Status, c.StatusCode, c.ResponseTime.Milliseconds(), c.Error, c.CheckedAt,
	)

	if err := row.Scan(&c.ID); err != nil {
		return models.Check{}, fmt.Errorf("inserting check: %w", err)
	}
	return c, nil
}

func (s *PostgresStorage) ListChecks(ctx context.Context, monitorID int64, limit int) ([]models.Check, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, monitor_id, status, status_code, response_time_ms, error, checked_at
		 FROM checks WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT $2`,
		monitorID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("querying checks: %w", err)
	}
	defer rows.Close()

	var checks []models.Check
	for rows.Next() {
		var c models.Check
		var responseTimeMs int64

		if err := rows.Scan(&c.ID, &c.MonitorID, &c.Status, &c.StatusCode, &responseTimeMs, &c.Error, &c.CheckedAt); err != nil {
			return nil, fmt.Errorf("scanning check row: %w", err)
		}
		c.ResponseTime = time.Duration(responseTimeMs) * time.Millisecond
		checks = append(checks, c)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterating check rows: %w", err)
	}
	return checks, nil
}
