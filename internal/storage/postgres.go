package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
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
	CREATE TABLE IF NOT EXISTS users (
		id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		name TEXT NOT NULL,
		email TEXT NOT NULL UNIQUE,
		password_hash TEXT NOT NULL,
		created_at TIMESTAMPTZ NOT NULL DEFAULT now()
	);

	CREATE TABLE IF NOT EXISTS monitors (
		id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
		user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
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

	CREATE INDEX IF NOT EXISTS idx_monitors_user_id ON monitors(user_id);
	CREATE INDEX IF NOT EXISTS idx_checks_monitor_id ON checks(monitor_id);
	CREATE INDEX IF NOT EXISTS idx_checks_monitor_time ON checks(monitor_id, checked_at DESC);`

	_, err := pool.Exec(ctx, schema)
	return err
}

func (s *PostgresStorage) CreateUser(ctx context.Context, u models.User) (models.User, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO users (name, email, password_hash) VALUES ($1, $2, $3) RETURNING id, created_at`, u.Name, u.Email, u.PasswordHash)
	if err := row.Scan(&u.ID, &u.CreatedAt); err != nil {
		return models.User{}, fmt.Errorf("inserting user: %w", err)
	}
	return u, nil
}

func (s *PostgresStorage) UpdateUserName(ctx context.Context, userID int64, name string) (models.User, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET name = $1 WHERE id = $2`, name, userID)
	if err != nil {
		return models.User{}, fmt.Errorf("updating user name: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return models.User{}, fmt.Errorf("user not found")
	}
	return s.GetUserByID(ctx, userID)
}

func (s *PostgresStorage) UpdateUserPassword(ctx context.Context, userID int64, newPasswordHash string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, newPasswordHash, userID)
	if err != nil {
		return fmt.Errorf("updating password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}

func (s *PostgresStorage) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	var u models.User
	row := s.pool.QueryRow(ctx, `SELECT id, name, email, password_hash, created_at FROM users WHERE email = $1`, email)

	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return models.User{}, fmt.Errorf("user not found: %w", err)
	}
	return u, nil
}

func (s *PostgresStorage) GetUserByID(ctx context.Context, id int64) (models.User, error) {
	var u models.User

	row := s.pool.QueryRow(ctx,
		`SELECT id, name, email, password_hash, created_at FROM users WHERE id = $1`, id)

	if err := row.Scan(&u.ID, &u.Name, &u.Email, &u.PasswordHash, &u.CreatedAt); err != nil {
		return models.User{}, fmt.Errorf("user not found: %w", err)
	}
	return u, nil
}

func (s *PostgresStorage) CreateMonitor(ctx context.Context, m models.Monitor) (models.Monitor, error) {
	row := s.pool.QueryRow(ctx,
		`INSERT INTO monitors (user_id, name, url, interval_seconds)
		 VALUES ($1, $2, $3, $4)
		 RETURNING id, created_at`,
		m.UserID, m.Name, m.URL, int(m.Interval.Seconds()),
	)
	if err := row.Scan(&m.ID, &m.CreatedAt); err != nil {
		return models.Monitor{}, fmt.Errorf("inserting monitor: %w", err)
	}
	return m, nil
}

func (s *PostgresStorage) UpdateMonitor(ctx context.Context, id, userID int64, name string, interval time.Duration) (models.Monitor, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE monitors SET name = $1, interval_seconds = $2 WHERE id = $3 AND user_id = $4`,
		name, int(interval.Seconds()), id, userID,
	)
	if err != nil {
		return models.Monitor{}, fmt.Errorf("updating monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return models.Monitor{}, fmt.Errorf("monitor not found")
	}

	return s.GetMonitorForUser(ctx, id, userID)
}

func (s *PostgresStorage) ListMonitorsForUser(ctx context.Context, userID int64) ([]models.Monitor, error) {
	return s.queryMonitors(ctx, "WHERE user_id = $1 ORDER BY id", userID)
}

func (s *PostgresStorage) ListAllMonitors(ctx context.Context) ([]models.Monitor, error) {
	return s.queryMonitors(ctx, "ORDER BY id")
}

func (s *PostgresStorage) queryMonitors(ctx context.Context, whereOrderClause string, args ...any) ([]models.Monitor, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, user_id, name, url, interval_seconds, created_at FROM monitors `+whereOrderClause,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("querying monitors: %w", err)
	}
	defer rows.Close()

	monitors := make([]models.Monitor, 0)
	for rows.Next() {
		var m models.Monitor
		var intervalSeconds int

		if err := rows.Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &intervalSeconds, &m.CreatedAt); err != nil {
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

func (s *PostgresStorage) GetMonitorForUser(ctx context.Context, id, userID int64) (models.Monitor, error) {
	var m models.Monitor
	var intervalSeconds int

	row := s.pool.QueryRow(ctx,
		`SELECT id, user_id, name, url, interval_seconds, created_at FROM monitors WHERE id = $1 AND user_id = $2`, id, userID)

	if err := row.Scan(&m.ID, &m.UserID, &m.Name, &m.URL, &intervalSeconds, &m.CreatedAt); err != nil {
		return models.Monitor{}, fmt.Errorf("monitor not found: %w", err)
	}
	m.Interval = time.Duration(intervalSeconds) * time.Second
	return m, nil
}

func (s *PostgresStorage) DeleteMonitorForUser(ctx context.Context, id, userID int64) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM monitors WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("deleting monitor: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("monitor not found")
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
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			return models.Check{}, ErrMonitorNotFound
		}
		return models.Check{}, fmt.Errorf("inserting check: %w", err)
	}
	return c, nil
}

func (s *PostgresStorage) ListChecks(ctx context.Context, monitorID int64, limit, offset int) ([]models.Check, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, monitor_id, status, status_code, response_time_ms, error, checked_at
		 FROM checks WHERE monitor_id = $1 ORDER BY checked_at DESC LIMIT $2 OFFSET $3`,
		monitorID, limit, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("querying checks: %w", err)
	}
	defer rows.Close()

	checks := make([]models.Check, 0)
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

func (s *PostgresStorage) CountChecks(ctx context.Context, monitorID int64) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM checks WHERE monitor_id = $1`, monitorID,
	).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting checks: %w", err)
	}
	return count, nil
}
